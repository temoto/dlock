# coding: utf-8
import pkg_resources

try:
    # cache
    VERSION = pkg_resources.get_distribution("dlock").version
except Exception:
    VERSION = "unknown"
__version__ = VERSION