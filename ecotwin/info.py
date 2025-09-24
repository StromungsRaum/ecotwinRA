import click
import pprintpp


@click.command(help="Information about the Twin system")
@click.pass_obj
def info(twin_info):
    """Display information about the EcoTwin system."""
    click.echo("EcoTwin System Information:")
    config = pprintpp.pformat(
        twin_info.config,
        indent=2,
        width=120,
        depth=4,
    )
    click.echo(config)
    # click.echo("Applications:")
    # pprintpp.pprint(twin_info.config["applications"], width=150, depth=20, indent=4)
    # click.echo("Models:")
    # pprintpp.pprint(twin_info.config["models"], width=150)
