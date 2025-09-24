import click


@click.command(help="Information about the Twin system")
@click.pass_obj
def info(twin_info):
    """Display information about the EcoTwin system."""
    click.echo("EcoTwin System Information:")
    click.echo(f"Config: {twin_info.config}")
