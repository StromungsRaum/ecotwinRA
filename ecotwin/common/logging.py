try:
    from loguru import logger
except ImportError:  # pragma: no cover
    import logging

    logger = logging.getLogger("ecotwin")

import sys


def prepare_logger() -> None:
    """Prepare logging."""
    if hasattr(logger, "remove"):
        # loguru
        logger.remove()
        format_str = "<green>{time:YYYY-MM-DD HH:mm:ss}</green> | <level>{level: <8}</level> | <level>{message}</level>"  # noqa:E501
        logger.add(
            sys.stdout, format=format_str, colorize=True, level="INFO", enqueue=True
        )
    else:
        # fallback to standard logging
        handler = logging.StreamHandler(sys.stdout)
        handler.setFormatter(
            logging.Formatter("%(asctime)s | %(levelname)-8s | %(message)s")
        )
        logger.addHandler(handler)
        logger.setLevel(logging.INFO)
