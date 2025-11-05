from pathlib import Path
import re

import click
from beeprint import pp
import yaml


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
def command(twin_info, output_format, root_name, output_dir):
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

    registry = _build_registry(config)
    context = {
        "config": config,
        "registry": registry,
        "output_dir": output_dir,
        "used_filenames": set(),
        "entity_files": {},
    }

    root_stem = _safe_filename(root_name, "root")
    root_filename = _uniq_filename(root_stem, context["used_filenames"])
    root_path = output_dir / root_filename

    lines = [f"# {root_name}"]
    processes = config.get("industrial_processes", {})

    if processes:
        lines.extend(["", "## Industrial Processes"])
        for process_id in processes:
            entity_info = _write_entity(process_id, context)
            if entity_info:
                filename, title = entity_info
                lines.append(f"- [{title}]({filename})")
            else:
                lines.append(f"- {process_id} (missing)")

    _write_markdown(root_path, lines)
    return root_path


def _build_registry(config):
    """Collect all entity definitions by ID."""
    registry = {}
    for section, entries in config.items():
        if not isinstance(entries, dict):
            continue
        for entity_id, data in entries.items():
            if isinstance(data, dict):
                registry[entity_id] = {"section": section, "data": data}
    return registry


def _write_entity(entity_id, context):
    """Create or return the markdown file for an entity."""
    cache = context["entity_files"]
    if entity_id in cache:
        return cache[entity_id]

    entity = context["registry"].get(entity_id)
    if not entity:
        click.echo(f"Warning: entity '{entity_id}' not defined.")
        return None

    data = entity["data"]
    title = data.get("name") or entity_id
    stem = _safe_filename(title, entity_id)
    filename = _uniq_filename(stem, context["used_filenames"])
    cache[entity_id] = (filename, title)

    lines = [f"# {title}"]
    description = data.get("description")
    if description:
        lines.extend(["", description])

    remaining = {
        key: value for key, value in data.items() if key not in {"name", "description"}
    }

    if remaining:
        lines.extend(["", "## Fields"])
        for key, value in remaining.items():
            lines.extend(_render_field(key, value))

    references = _collect_references(remaining, context["registry"])
    references.discard(entity_id)

    if references:
        lines.extend(["", "## Linked Entities"])
        by_section = {}
        for ref_id in references:
            ref_entity = context["registry"].get(ref_id)
            if not ref_entity:
                continue
            section = ref_entity["section"]
            by_section.setdefault(section, []).append(ref_id)

        for section, ref_ids in sorted(by_section.items()):
            section_title = section.replace("_", " ").title()
            lines.append(f"### {section_title}")
            for ref_id in sorted(
                ref_ids,
                key=lambda rid: context["registry"][rid]["data"].get("name") or rid,
            ):
                ref_info = _write_entity(ref_id, context)
                if ref_info:
                    ref_filename, ref_title = ref_info
                    lines.append(f"- [{ref_title}]({ref_filename})")
                else:
                    lines.append(f"- {ref_id} (missing)")

    _write_markdown(context["output_dir"] / filename, lines)
    return cache[entity_id]


def _render_field(key, value):
    """Render a field key/value pair into markdown lines."""
    label = f"- **{key}**"

    if isinstance(value, dict) or isinstance(value, list):
        serialized = yaml.safe_dump(value, sort_keys=False).rstrip()
        if not serialized:
            return [f"{label}: []"]
        return [label + ":", "", "```yaml", serialized, "```"]

    return [f"{label}: {value}"]


def _collect_references(value, registry):
    """Recursively collect entity references from a nested structure."""
    references = set()

    if isinstance(value, str):
        if value in registry:
            references.add(value)
        return references

    if isinstance(value, dict):
        for key, item in value.items():
            if key == "name":
                continue
            references.update(_collect_references(item, registry))
        return references

    if isinstance(value, (list, tuple, set)):
        for item in value:
            references.update(_collect_references(item, registry))
        return references

    return references
