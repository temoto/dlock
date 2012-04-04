# coding: utf-8
from distribute_setup import use_setuptools
use_setuptools()
from setuptools import setup

from get_git_version import get_git_version


REQUIRES = (
    "gevent>=1.0b1",
    "tnetstring",
)


setup(
    name="dlock",
    version=get_git_version("dlock/VERSION", 6),
    description=u"Distributed lock manager based on TCP connections",
    author="Sergey Shepelev",
    author_email="temotor@gmail.com",
    url="https://github.com/temoto/dlock",

    install_requires=REQUIRES,
    packages=("dlock",),
    zip_safe=False,
    entry_points={
        "console_scripts": (
            "dlock = dlock.client:main",
            "dlock-server = dlock.server:main",
        ),
    },

    classifiers=(
        "Development Status :: 4 - Beta",
        "Environment :: Console",
        "Intended Audience :: Developers",
        "Intended Audience :: System Administrators",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 2",
    ),
)
