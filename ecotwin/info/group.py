import typer
from ecotwin.info.system import system_command
from ecotwin.info.models import models_command

app = typer.Typer(help="System Information", pretty_exceptions_enable=False)


@app.callback(invoke_without_command=False)
def info_callback(ctx: typer.Context) -> None:
    """Display information about the EcoTwin system."""
    pass


app.command("system")(system_command)
app.command("models")(models_command)
