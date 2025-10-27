from pathlib import Path
import re

import click
from beeprint import pp


def _format_tree(value, indent=0):
    """Render nested structures as a simple indented tree."""
    prefix = " " * indent

    if isinstance(value, dict):
        lines = []
        for key, item in value.items():
            if isinstance(item, (dict, list)):
                lines.append(f"{prefix}- {key}:")
                lines.extend(_format_tree(item, indent + 2))
            else:
                lines.append(f"{prefix}- {key}: {item}")
        return lines

    if isinstance(value, list):
        lines = []
        for index, item in enumerate(value):
            if isinstance(item, (dict, list)):
                lines.append(f"{prefix}- [{index}]")
                lines.extend(_format_tree(item, indent + 2))
            else:
                lines.append(f"{prefix}- [{index}]: {item}")
        return lines

    return [f"{prefix}{value}"]


def render_tree(value):
    """Return the tree formatted string for a nested structure."""
    return "\n".join(_format_tree(value))


def _format_markdown_tree(value, indent=0):
    """Render nested structures as a Markdown bullet list."""
    prefix = "  " * indent

    if isinstance(value, dict):
        lines = []
        for key, item in value.items():
            if isinstance(item, (dict, list)):
                lines.append(f"{prefix}- **{key}**")
                lines.extend(_format_markdown_tree(item, indent + 1))
            else:
                lines.append(f"{prefix}- **{key}**: {item}")
        return lines

    if isinstance(value, list):
        lines = []
        for index, item in enumerate(value):
            label = f"[{index}]"
            if isinstance(item, (dict, list)):
                lines.append(f"{prefix}- {label}")
                lines.extend(_format_markdown_tree(item, indent + 1))
            else:
                lines.append(f"{prefix}- {label}: {item}")
        return lines

    return [f"{prefix}- {value}"]


def render_markdown_tree(value):
    """Return the Markdown tree formatted string for a nested structure."""
    return "\n".join(_format_markdown_tree(value))


@click.command(help="Information about the Twin system")
@click.option(
    "--format",
    "output_format",
    type=click.Choice(
        ["raw", "tree", "markdown-tree", "markdown-files"], case_sensitive=False
    ),
    default="raw",
    show_default=True,
    help=(
        "Select raw beeprint output, tree hierarchy view, Markdown tree, "
        "or generate Markdown files."
    ),
)
@click.option(
    "--root-name",
    type=str,
    help="Name of the root Markdown document when generating files.",
)
@click.option(
    "--output-dir",
    type=click.Path(
        file_okay=False,
        dir_okay=True,
        resolve_path=True,
        path_type=Path,
    ),
    help="Directory where generated Markdown files will be written.",
)
@click.pass_obj
def info(twin_info, output_format, root_name, output_dir):
    """Display information about the EcoTwin system."""
    fmt = output_format.lower()
    if fmt == "tree":
        click.echo("EcoTwin System Information (tree):")
        click.echo(render_tree(twin_info.config))
    elif fmt == "markdown-tree":
        click.echo("EcoTwin System Information (markdown tree):")
        click.echo(render_markdown_tree(twin_info.config))
    elif fmt == "markdown-files":
        if not root_name:
            raise click.BadParameter(
                "root name is required when using markdown-files format.",
                param_hint="--root-name",
            )
        target_dir = output_dir or Path.cwd()
        root_path = generate_markdown_files(
            twin_info.config,
            root_name=root_name,
            output_dir=target_dir,
        )
        click.echo(f"Generated Markdown hierarchy rooted at {root_path}")
    else:
        click.echo("EcoTwin System Information:")
        pp(twin_info.config, indent=4)


def _safe_filename(value: str, fallback: str = "item") -> str:
    """Return a safe filename stem preserving case where possible."""
    candidate = value.strip() if value else fallback
    candidate = re.sub(r"[^A-Za-z0-9 _-]+", "", candidate)
    candidate = candidate.strip().replace(" ", "-")
    return candidate or fallback


def _uniq_filename(stem: str, used: set[str]) -> str:
    """Ensure filename uniqueness by appending numeric suffix if needed."""
    suffix = 1
    candidate = f"{stem}.md"
    lower_candidate = candidate.lower()
    while lower_candidate in used:
        candidate = f"{stem}-{suffix}.md"
        lower_candidate = candidate.lower()
        suffix += 1
    used.add(lower_candidate)
    return candidate


def _write_markdown(path: Path, lines: list[str]) -> None:
    """Write markdown content ensuring trailing newline."""
    content = "\n".join(lines).rstrip() + "\n"
    path.write_text(content, encoding="utf-8")


def generate_markdown_files(config, root_name: str, output_dir: Path) -> Path:
    """Generate Markdown files representing the twin hierarchy."""
    output_dir.mkdir(parents=True, exist_ok=True)
    context = {
        "config": config,
        "output_dir": output_dir,
        "used_filenames": set(),
        "process_files": {},
        "digital_twin_files": {},
        "application_files": {},
        "model_files": {},
    }

    root_stem = _safe_filename(root_name, "root")
    root_filename = _uniq_filename(root_stem, context["used_filenames"])
    root_path = output_dir / root_filename

    lines = [f"# {root_name}"]
    processes = config.get("industrial_processes", {})

    if processes:
        lines.extend(["", "## Industrial Processes"])
        for process_id, process_data in processes.items():
            process_filename, process_title = _write_process_file(
                process_id, process_data, context
            )
            if process_filename:
                lines.append(f"- [{process_title}]({process_filename})")

    _write_markdown(root_path, lines)
    return root_path


def _write_process_file(process_id, process_data, context):
    title = process_data.get("name") or process_id
    filename = context["process_files"].get(process_id)
    if filename:
        return filename, title

    stem = _safe_filename(title, process_id)
    filename = _uniq_filename(stem, context["used_filenames"])
    context["process_files"][process_id] = filename

    lines = [f"# {title}"]
    description = process_data.get("description")
    if description:
        lines.extend(["", description])

    digital_refs = process_data.get("digital_twin") or []
    if isinstance(digital_refs, str):
        digital_refs = [digital_refs]

    if digital_refs:
        lines.extend(["", "## Digital Twins"])
        for digital_id in digital_refs:
            digital_info = _write_digital_twin_file(digital_id, context)
            if digital_info:
                digital_filename, digital_title = digital_info
                lines.append(f"- [{digital_title}]({digital_filename})")
            else:
                lines.append(f"- {digital_id} (missing)")

    _write_markdown(context["output_dir"] / filename, lines)
    return filename, title


def _write_digital_twin_file(digital_id, context):
    cache = context["digital_twin_files"]
    if digital_id in cache:
        return cache[digital_id]

    digital_twins = context["config"].get("digital_twins", {})
    digital_data = digital_twins.get(digital_id)
    if not digital_data:
        click.echo(f"Warning: digital twin '{digital_id}' not defined.")
        return None

    title = digital_data.get("name") or digital_id
    stem = _safe_filename(title, digital_id)
    filename = _uniq_filename(stem, context["used_filenames"])
    cache[digital_id] = (filename, title)

    lines = [f"# {title}"]
    description = digital_data.get("description")
    if description:
        lines.extend(["", description])

    applications = digital_data.get("applications") or []
    if applications:
        lines.extend(["", "## Applications"])
        for app_id in applications:
            app_info = _write_application_file(app_id, context)
            if app_info:
                app_filename, app_title = app_info
                lines.append(f"- [{app_title}]({app_filename})")
            else:
                lines.append(f"- {app_id} (missing)")

    _write_markdown(context["output_dir"] / filename, lines)
    return filename, title


def _write_application_file(app_id, context):
    cache = context["application_files"]
    if app_id in cache:
        return cache[app_id]

    applications = context["config"].get("applications", {})
    app_data = applications.get(app_id)
    if not app_data:
        click.echo(f"Warning: application '{app_id}' not defined.")
        return None

    title = app_data.get("name") or app_id
    stem = _safe_filename(title, app_id)
    filename = _uniq_filename(stem, context["used_filenames"])
    cache[app_id] = (filename, title)

    lines = [f"# {title}"]
    description = app_data.get("description")
    if description:
        lines.extend(["", description])

    app_type = app_data.get("type")
    details = []
    if app_type:
        details.append(f"- Type: {app_type}")

    if details:
        lines.extend(["", "## Details"])
        lines.extend(details)

    models = app_data.get("model")
    if models:
        if not isinstance(models, (list, tuple, set)):
            models = [models]
        lines.extend(["", "## Models"])
        for model_id in models:
            model_info = _write_model_file(model_id, context)
            if model_info:
                model_filename, model_title = model_info
                lines.append(f"- [{model_title}]({model_filename})")
            else:
                lines.append(f"- {model_id} (missing)")

    _write_markdown(context["output_dir"] / filename, lines)
    return filename, title


def _write_model_file(model_id, context):
    cache = context["model_files"]
    if model_id in cache:
        return cache[model_id]

    models = context["config"].get("models", {})
    model_data = models.get(model_id)
    if not model_data:
        click.echo(f"Warning: model '{model_id}' not defined.")
        return None

    title = model_data.get("name") or model_id
    stem = _safe_filename(title, model_id)
    filename = _uniq_filename(stem, context["used_filenames"])
    cache[model_id] = (filename, title)

    lines = [f"# {title}"]
    description = model_data.get("description")
    if description:
        lines.extend(["", description])

    model_type = model_data.get("type")
    details = []
    if model_type:
        details.append(f"- Type: {model_type}")

    if details:
        lines.extend(["", "## Details"])
        lines.extend(details)

    _write_markdown(context["output_dir"] / filename, lines)
    return filename, title
