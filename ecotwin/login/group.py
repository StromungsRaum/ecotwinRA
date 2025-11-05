import click

from ecotwin.common.lazy_group import LazyGroup


@click.group(
    cls=LazyGroup,
    lazy_subcommands={"store": "ecotwin.login.store.store"},
    help="Login to StrömungsRaum",
)
@click.pass_obj
def command(twin_info):
    """Login to the EcoTwin system."""
    print("login")
