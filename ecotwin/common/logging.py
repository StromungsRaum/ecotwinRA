from loguru import logger
import sys


def prepare_logger() -> None:
    """Prepare logging."""
    logger.remove()

    format_str = "<green>{time:YYYY-MM-DD HH:mm:ss}</green> | <level>{level: <8}</level> | <level>{message}</level>"  # noqa:E501
    logger.add(sys.stdout, format=format_str, colorize=True, level="INFO", enqueue=True)

    format_str = "<red>{time:YYYY-MM-DD HH:mm:ss}</red> | <level>{level: <8}</level> | <level>{message}</level>"  # noqa:E501
    logger.add(sys.stderr, format=format_str, colorize=True, level="INFO")
