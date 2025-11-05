#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "beeprint",
#     "click",
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
# ecotwin = { path = "../", editable = true }
# ///

from webbrowser import get
import click

from pathlib import Path
from ecotwin.common.lazy_group import LazyGroup
from ecotwin.common.twin_config import read_twin_yaml
from loguru import logger
from ecotwin.common.logging import prepare_logger


@click.group(
    cls=LazyGroup,
    lazy_subcommands={
        "info": "ecotwin.info.group.command",
        "login": "ecotwin.login.group.command",
    },
    help="Twin Control",
)
@click.version_option(package_name="ecotwin")
@click.option("--file", help="YAML file to use", type=click.Path(exists=True))
@click.pass_context
def twinctl(ctx, file):
    """EcoTwin Twin Control CLI."""
    prepare_logger()

    if file is not None:
        file = Path(file)
    else:
        file = Path("twin.yaml")
    logger.info(f"Using config file: {file}")

    ctx.obj = read_twin_yaml(file)


if __name__ == "__main__":
    twinctl()
