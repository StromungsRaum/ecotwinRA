#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "beeprint",
#     "typer[all]",
#     "dotenv",
#     "ecotwin",
#     "loguru",
#     "pyyaml",
#     "requests",
#     "requests-toolbelt",
#     "urllib3",
# ]
#
# [tool.uv.sources]
# ecotwin = { path = "../../", editable = true }
# ///

import typer

from ecotwin.common.lazy_typer_group import LazyTyperGroup
from ecotwin.common.logging import prepare_logger

# Create the main app
app = typer.Typer(
    help="Twin Control",
    context_settings={"help_option_names": ["-h", "--help"]},
    pretty_exceptions_short=True,
    pretty_exceptions_enable=False,
)


def version_callback(value: bool) -> None:
    """Handle --version flag."""
    if value:
        try:
            from importlib.metadata import version

            typer.echo(f"ecotwin, version {version('ecotwin')}")
        except Exception:
            typer.echo("ecotwin, version unknown")
        raise typer.Exit()


@app.callback()
def main(
    version: bool = typer.Option(
        None,
        "--version",
        callback=version_callback,
        is_eager=True,
        help="Show version and exit",
    ),
) -> None:
    """EcoTwin Twin Control CLI."""


# Create lazy loader for subcommands
lazy_loader = LazyTyperGroup(
    app,
    lazy_subcommands={
        "info": "ecotwin.info.group.app",
        "login": "ecotwin.login.store.app",
    },
)

# Eagerly load subcommands to ensure they're added
lazy_loader.load_all()


if __name__ == "__main__":
    prepare_logger()

    app()
