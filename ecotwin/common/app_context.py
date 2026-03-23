"""Application context and state management."""

# Global application context
_app_context = {
    "twin_info": None,
}


def set_twin_info(twin_info) -> None:
    """Set the twin info context."""
    _app_context["twin_info"] = twin_info


def get_twin_info():
    """Get the twin info context."""
    return _app_context.get("twin_info")


def clear_context() -> None:
    """Clear the application context."""
    _app_context["twin_info"] = None
