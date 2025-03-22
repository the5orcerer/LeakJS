import os
import sys

try:
    from setuptools import setup, find_packages
except ImportError:
    os.system(f"{sys.executable} -m pip install setuptools")
    from setuptools import setup, find_packages

# Read the version from the version.txt file
def read_version():
    with open("version.txt", "r") as file:
        return file.read().strip()

# Read the long description from the README.md file
def read_long_description():
    with open("README.md", "r", encoding="utf-8") as file:
        return file.read()

setup(
    name="LeakJS",
    version=read_version(),
    description="A JavaScript leak detection tool",
    long_description=read_long_description(),
    long_description_content_type="text/markdown",
    author="rootplinix",
    author_email="the5orc3rer@gmail.com",
    url="https://github.com/the5orcerer/LeakJS",
    packages=find_packages(),
    include_package_data=True,
    install_requires=[
        "aiohttp",
        "aiohttp_retry",
        "argparse",
        "colorama",
        "tqdm",
        "pyyaml"
    ],
    entry_points={
        "console_scripts": [
            "leakjs=main:main",
        ],
    },
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires=">=3.6",
)
