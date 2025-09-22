#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "click",
#     "ecotwin",
# ]
#
# [tool.uv.sources]
# ecotwin = { path = "../", editable = true }
# ///

import click

from ecotwin.lazy_group import LazyGroup


@click.group(
    cls=LazyGroup,
    lazy_subcommands={"info": "ecotwin.info.info"},
    help="Twin Control",
)
@click.version_option(package_name="ecotwin")
def twinctl():
    print("Hello from twinctl!")


if __name__ == "__main__":
    twinctl()
