import click


@click.command(help="Information about the Twin system")
def info():
    """Display information about the EcoTwin system."""
    click.echo("EcoTwin System Information:")
