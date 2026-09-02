import typer

from ecotwin.submission.submit import submit_command

app = typer.Typer(help="Submit models to StrömungsRaum", pretty_exceptions_enable=False)


@app.callback(invoke_without_command=False)
def submission_callback(ctx: typer.Context) -> None:
    """Submit full models (geometry + digital twin + simulation) to the EcoTwin system."""
    pass


app.command("submit")(submit_command)
