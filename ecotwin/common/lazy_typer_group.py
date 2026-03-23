"""Lazy loading support for Typer apps.

Inspired by Click's LazyGroup for deferred module loading.
"""

import importlib
from typing import Dict, Optional
import typer


class LazyTyperGroup:
    """Wrapper for a Typer app that supports lazy-loading of subcommands.

    Subcommands are only imported when they're first accessed, reducing startup time.
    """

    def __init__(
        self, app: typer.Typer, lazy_subcommands: Optional[Dict[str, str]] = None
    ):
        """Initialize the lazy group.

        Args:
            app: The Typer application
            lazy_subcommands: Mapping of command name to import path (e.g., "module.submodule.command")
        """
        self.app = app
        self.lazy_subcommands = lazy_subcommands or {}
        self._loaded: Dict[str, typer.Typer] = {}

    def get_command(self, cmd_name: str) -> Optional[typer.Typer]:
        """Get a command, loading it lazily if needed."""
        if cmd_name in self._loaded:
            return self._loaded[cmd_name]

        if cmd_name in self.lazy_subcommands:
            return self._lazy_load(cmd_name)

        return None

    def _lazy_load(self, cmd_name: str) -> typer.Typer:
        """Lazily load a command by importing its module."""
        import_path = self.lazy_subcommands[cmd_name]
        modname, cmd_object_name = import_path.rsplit(".", 1)

        # Import the module
        mod = importlib.import_module(modname)

        # Get the command object from that module
        cmd_object = getattr(mod, cmd_object_name)

        # Validate it's a Typer app
        if not isinstance(cmd_object, typer.Typer):
            raise ValueError(
                f"Lazy loading of {import_path} failed: "
                f"expected typer.Typer instance, got {type(cmd_object)}"
            )

        # Cache it
        self._loaded[cmd_name] = cmd_object

        # Add it to the app
        self.app.add_typer(cmd_object, name=cmd_name)

        return cmd_object

    def load_all(self) -> None:
        """Eagerly load all lazy subcommands."""
        for cmd_name in list(self.lazy_subcommands.keys()):
            if cmd_name not in self._loaded:
                self._lazy_load(cmd_name)
