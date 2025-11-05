import click

from ecotwin.common.lazy_group import LazyGroup


@click.group(
    cls=LazyGroup,
    lazy_subcommands={
        "system": "ecotwin.info.system.command",
        "models": "ecotwin.info.models.command",
    },
    help="System Information",
)
@click.pass_obj
def command(twin_info):
    """Display information about the EcoTwin system."""
    pass
