import click


@click.command(help="Information about the Twin system")
def info():
    """Display information about the EcoTwin system."""
    click.echo("EcoTwin System Information:")
    click.echo("Version: 1.0.0")
    click.echo("Status: Operational")
