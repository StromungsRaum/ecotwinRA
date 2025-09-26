import click
from beeprint import pp


@click.command(help="Information about the Twin system")
@click.pass_obj
def info(twin_info):
    """Display information about the EcoTwin system."""
    click.echo("EcoTwin System Information:")
    pp(twin_info.config, indent=4)
