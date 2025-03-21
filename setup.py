from setuptools import setup, find_packages

setup(
    name="LeakJS",
    version="1.0.0",
    description="A tool for scanning JavaScript files and URLs for patterns.",
    author="the5orcerer",
    author_email="fermion.farmin77@gmail.com",
    url="https://github.com/the5orcerer/LeakJS",
    packages=find_packages(),
    install_requires=[
        "aiohttp==3.8.1",
        "aiohttp_retry==2.4.6",
        "colorama==0.4.4",
        "requests==2.26.0",
        "PyYAML==5.4.1",
        "tqdm==4.62.3",
    ],
    entry_points={
        'console_scripts': [
            'leakjs=main:main',
        ],
    },
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
    python_requires='>=3.6',
)