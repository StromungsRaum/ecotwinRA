#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "beeprint",
#     "click",
#     "ecotwin",
#     "pprintpp",
#     "pyyaml",
# ]
#
# [tool.uv.sources]
# ecotwin = { path = "../", editable = true }
# ///

import click

from pathlib import Path
from ecotwin.lazy_group import LazyGroup
from ecotwin.twin_info import read_twin_yaml


@click.group(
    cls=LazyGroup,
    lazy_subcommands={"info": "ecotwin.info.info"},
    help="Twin Control",
)
@click.version_option(package_name="ecotwin")
@click.option("--file", help="YAML file to use", type=click.Path(exists=True))
@click.pass_context
def twinctl(ctx, file):
    if file is not None:
        file = Path(file)
    else:
        file = Path("twin.yaml")
    print(f"Using config file: {file}")
    ctx.obj = read_twin_yaml(file)


if __name__ == "__main__":
    twinctl()
