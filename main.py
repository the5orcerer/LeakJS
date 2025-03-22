#!/bin/python3
import re
import asyncio
import aiohttp
import yaml
import argparse
import sys
import signal
from aiohttp import ClientSession
from aiohttp_retry import RetryClient, ExponentialRetry
from queue import Queue
from typing import List, Dict, Any, Optional
from colorama import Fore, Style, init
from tqdm import tqdm
import logging
import os
import subprocess

init(autoreset=True)

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

CONFIG="~/.config/LeakJS"
tool_url = "https://github.com/the5orcerer/LeakJS"
VERSION_FILE = f"{CONFIG}/version.txt"
REMOTE_VERSION_URL = "https://raw.githubusercontent.com/the5orcerer/LeakJS/main/version.txt"
REGEX_REPO = "https://github.com/the5orcerer/Bishop"

def get_local_version() -> str:
    """Get the local version from the version.txt file."""
    try:
        version_file = os.path.expanduser(VERSION_FILE)
        with open(version_file, 'r') as file:
            return file.read()
    except FileNotFoundError as e:
        return e

def banner():
    init(autoreset=True)
    version = get_local_version()
    print(f"""
                        __    v{version}     __       _______
                       / /   ___  ____ _/ /__    / / ___/
                      / /   / _ \\/ __ `/ //_/_  / /\\__ \\ 
                     / /___/  __/ /_/ / ,< / /_/ /___/ / 
                    /_____/\\___/\\__,_/_/|_|\\____//____/  
                                               {Style.BRIGHT}{Fore.RED}@rootplinix
            """)

def read_urls(file_path: str) -> List[str]:
    """Read URLs from a file and return a sorted list of .js URLs."""
    try:
        with open(file_path, 'r') as file:
            urls = [line.strip() for line in file if line.strip().endswith('.js')]
        return sorted(urls)
    except FileNotFoundError:
        print(f"[{Fore.RED} ERR {Style.RESET_ALL}] File not found: {file_path}")

def read_patterns(yaml_file: str) -> List[Dict[str, Any]]:
    """Read patterns from a YAML file."""
    try:
        with open(yaml_file, 'r') as file:
            data = yaml.safe_load(file)
        return data['patterns']
    except FileNotFoundError:
        print(f"[{Fore.RED} ERR {Style.RESET_ALL}] Regex file not found: {yaml_file}")

def load_default_patterns() -> List[Dict[str, Any]]:
    """Load default patterns from the .config/LeakJS directory."""
    patterns = []
    config_dir = os.path.expanduser(CONFIG)
    if os.path.exists(config_dir) and os.path.isdir(config_dir):
        for file_name in os.listdir(config_dir):
            if file_name.endswith(".yaml"):
                file_path = os.path.join(config_dir, file_name)
                patterns.extend(read_patterns(file_path))
    return patterns

def parse_direct_patterns(patterns_str: str) -> List[Dict[str, Any]]:
    """Parse directly inputted regex patterns."""
    patterns = []
    for pattern in patterns_str.split(';'):
        if pattern.strip():
            try:
                re.compile(pattern.strip())
                patterns.append({
                    'pattern': {
                        'name': 'DirectPattern',
                        'regex': pattern.strip(),
                        'confidence': 'Unknown'
                    }
                })
            except re.error:
                logging.warning(f"Invalid regex pattern: {pattern.strip()} - Skipped")
    return patterns

def read_file_content(file_path: str) -> str:
    """Read the content of a file."""
    try:
        with open(file_path, 'r') as file:
            return file.read()
    except FileNotFoundError:
        print(f"[{Fore.RED} ERR {Style.RESET_ALL}] File not found: {file_path}")
        return ""
    except IOError as e:
        print(f"[{Fore.RED} ERR {Style.RESET_ALL}] Error reading file {file_path}: {e}")
        return ""

def search_patterns(content: str, patterns: List[Dict[str, Any]]) -> Dict[str, Dict[str, Any]]:
    """Search for patterns in the content."""
    results = {}
    for pattern in patterns:
        name = pattern['pattern']['name']
        regex = pattern['pattern']['regex']
        confidence = pattern['pattern']['confidence']
        matches = re.findall(regex, content)
        if matches:
            results[name] = {'matches': matches, 'confidence': confidence}
    return results

def display_results(source: str, results: Dict[str, Dict[str, Any]], source_type: str = "URL", verbose: bool = False) -> None:
    """Display the results of pattern matching."""
    if results:
        print(f"[{Fore.GREEN} SUCC {Style.RESET_ALL}] {source} ({source_type})\n")
        for name, data in results.items():
            print(f"Regex name: {name}")
            for match in data['matches']:
                print(f"{match}")
            print(f"Confidence: {data['confidence']}\n")

def save_results_to_file(output_file: str, source: str, results: Dict[str, Dict[str, Any]], source_type: str = "URL") -> None:
    """Save the results of pattern matching to a file."""
    with open(output_file, 'a') as file:
        file.write(f"[ SUCC ] {source} ({source_type})\n")
        for name, data in results.items():
            file.write(f"{name}\n")
            for match in data['matches']:
                file.write(f"{match}\n")
            file.write(f"Confidence: {data['confidence']}\n")
        file.write("\n")

async def fetch_url_content(url: str, session: ClientSession, verbose: bool) -> str:
    """Fetch the content of a URL with detailed HTTP headers."""
    headers = {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'Accept-Language': 'en-US,en;q=0.5',
        'Connection': 'keep-alive',
        'Upgrade-Insecure-Requests': '1',
    }
    try:
        async with session.get(url, headers=headers) as response:
            response.raise_for_status()
            return await response.text()
    except aiohttp.ClientError as e:
        if verbose:
            logging.error(f"Error fetching URL {url}: {e}")
        return ""

async def process_url(url: str, patterns: List[Dict[str, Any]], session: ClientSession, output_file: Optional[str] = None, verbose: bool = False, silent: bool = False) -> None:
    """Process a single URL to fetch content and search for patterns."""
    content = await fetch_url_content(url, session, verbose)
    if content:
        results = search_patterns(content, patterns)
        if results:
            if output_file:
                save_results_to_file(output_file, url, results)
            if not silent:
                display_results(url, results, verbose=verbose)

async def process_file(file_path: str, patterns: List[Dict[str, Any]], output_file: Optional[str] = None, verbose: bool = False, silent: bool = False) -> None:
    """Process a single file to read content and search for patterns."""
    content = read_file_content(file_path)
    if content:
        results = search_patterns(content, patterns)
        if results:
            if output_file:
                save_results_to_file(output_file, file_path, results, source_type="File")
            if not silent:
                display_results(file_path, results, source_type="File", verbose=verbose)

async def worker(queue: Queue, patterns: List[Dict[str, Any]], source_type: str, output_file: Optional[str] = None, verbose: bool = False, silent: bool = False) -> None:
    """Worker function for threading."""
    async with RetryClient(raise_for_status=False) as session:
        while not queue.empty():
            source = queue.get()
            if source_type == "URL":
                await process_url(source, patterns, session, output_file, verbose, silent)
            elif source_type == "File":
                await process_file(source, patterns, output_file, verbose, silent)
            queue.task_done()

async def run_leakjs(urls_file: Optional[str], single_url: Optional[str], patterns_file: Optional[str], direct_patterns: Optional[str], file_path: Optional[str], concurrency: int, output_file: Optional[str], verbose: bool, silent: bool) -> None:
    """Main function to handle input and start the scanning process."""
    patterns = load_default_patterns()
    if patterns_file:
        patterns.extend(read_patterns(patterns_file))
    if direct_patterns:
        patterns.extend(parse_direct_patterns(direct_patterns))

    print(f"[{Fore.BLUE} INF {Style.RESET_ALL}] Templates loaded: {len(patterns)}\n")

    urls = []
    if urls_file:
        urls = read_urls(urls_file)
    elif single_url:
        urls = [single_url]
    elif not sys.stdin.isatty():
        urls = [line.strip() for line in sys.stdin if line.strip().endswith('.js')]

    if not urls and not file_path:
        print(f"[{Fore.RED} ERR {Style.RESET_ALL}] No URLs or files provided.")
        return

    if silent and not output_file:
        print(f"[{Fore.RED} ERR {Style.RESET_ALL}] Output file must be specified in silent mode (-s)")
        return

    queue = Queue()
    source_type = "URL"
    if urls:
        for url in urls:
            queue.put(url)
    if file_path:
        queue.put(file_path)
        source_type = "File"

    if silent:
        with tqdm(total=queue.qsize(), desc="Processing") as pbar:
            tasks = []
            for _ in range(concurrency):
                task = asyncio.create_task(worker(queue, patterns, source_type, output_file, verbose, silent))
                tasks.append(task)

            while not queue.empty():
                queue.get()
                pbar.update(1)
                queue.task_done()

            await asyncio.gather(*tasks)
    else:
        tasks = []
        for _ in range(concurrency):
            task = asyncio.create_task(worker(queue, patterns, source_type, output_file, verbose, silent))
            tasks.append(task)

        await asyncio.gather(*tasks)

def update_tool() -> None:
    """Function to update the tool automatically."""
    subprocess.run(["git", "clone", tool_url], check=True)
    subprocess.run(["cd", "LeakJS"])
    subprocess.run(["python3", "install.py"])
    print(f"[{Fore.GREEN} SUCC {Style.RESET_ALL}] LeakJS updated successfully")

def download_regexes(repo_url: str) -> None:
    """Download regexes from a repository into .config/LeakJS."""
    config_dir = os.path.expanduser(CONFIG)
    if not os.path.exists(config_dir):
        os.makedirs(config_dir)
    subprocess.run(["git", "clone", repo_url, config_dir], check=True)
    print(f"[{Fore.GREEN} SUCC {Style.RESET_ALL}] Regexes downloaded successfully")

    # Add to bashrc or fish_config for tab support
    shell = os.getenv('SHELL')
    if 'bash' in shell:
        bashrc_path = os.path.expanduser("~/.bashrc")
        with open(bashrc_path, 'a') as bashrc:
            bashrc.write(f'\ncomplete -W "$(ls {config_dir})" leakjs\n')
        subprocess.run(["source", bashrc_path], check=True, shell=True)
    elif 'fish' in shell:
        fish_config_path = os.path.expanduser("~/.config/fish/config.fish")
        with open(fish_config_path, 'a') as fish_config:
            fish_config.write(f'\ncomplete -c leakjs -f -a "(ls {config_dir})"\n')
        subprocess.run(["source", fish_config_path], check=True, shell=True)

    print(f"[{Fore.GREEN} SUCC {Style.RESET_ALL}]Tab completion configured successfully")

def update_templates() -> None:
    """Function to update templates."""
    download_regexes(REGEX_REPO)
    print(f"[{Fore.GREEN} SUCC {Style.RESET_ALL}]Templates updated successfully")

async def get_remote_version() -> str:
    """Get the remote version from the GitHub repository."""
    async with aiohttp.ClientSession() as session:
        try:
            async with session.get(REMOTE_VERSION_URL) as response:
                response.raise_for_status()
                text = await response.text()
                return text.strip()
        except aiohttp.ClientError:
            return get_local_version()

async def check_version() -> None:
    """Check the current version against the remote version."""
    local_version = get_local_version()
    remote_version = await get_remote_version()
    if local_version == remote_version:
        print(f"[{Fore.BLUE} INF {Style.RESET_ALL}] Current LeakJS version v{local_version} [updated]")
    else:
        print(f"[{Fore.RED} ERR {Style.RESET_ALL}] Current LeakJS version v{local_version} [outdated]")
        print(f"[{Fore.BLUE} INF {Style.RESET_ALL}] Latest LeakJS version v{remote_version} available. Run with -up to update.")

def print_help() -> None:
    """Print help message."""
    help_text = """
    LeakJS - A JavaScript leak detection tool

    Usage: leakjs [options]

    Options:
      -u, --url         Single URL to scan
      -l, --list        Path to the file containing URLs (one per line)
      -p, --patterns    Path to the YAML file containing patterns (optional)
      -r, --regex       Directly input regex patterns separated by ";"
      -f, --file        Path to a JavaScript file to scan
      -c, --concurrency Number of concurrent threads (default: 1)
      -o, --output      Path to the output file to save results
      -v, --verbose     Enable verbose logging
      -s, --silent      Show progress bar without any output in the terminal
      -t, --threads     Number of threads to use
      -up, --update     Update the tool automatically
      -upt, --updatetemplates Update the templates
      -h, --help        Show this help message and exit

    Examples:
      leakjs -u http://example.com/app.js
      leakjs -l urls.txt -p patterns.yaml
      leakjs -r "regex1;regex2" -f app.js -c 5
      leakjs -up
    """
    print(help_text)

def handle_sigint(signal, frame):
    print(f"\n[{Fore.RED} ERR {Style.RESET_ALL}] Process interrupted. Exiting gracefully!")
    sys.exit(0)

def main():
    banner()
    signal.signal(signal.SIGINT, handle_sigint)

    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument('-u', '--url', type=str, help='Single URL to scan')
    parser.add_argument('-l', '--list', type=str, help='Path to the file containing URLs')
    parser.add_argument('-p', '--patterns', type=str, help='Path to the YAML file containing patterns (optional)')
    parser.add_argument('-r', '--regex', type=str, help='Directly input regex patterns separated by ";"')
    parser.add_argument('-f', '--file', type=str, help='Path to a JavaScript file to scan')
    parser.add_argument('-c', '--concurrency', type=int, default=1, help='Number of concurrent threads (default: 1)')
    parser.add_argument('-o', '--output', type=str, help='Path to the output file to save results')
    parser.add_argument('-v', '--verbose', action='store_true', help='Enable verbose logging')
    parser.add_argument('-s', '--silent', action='store_true', help='Show progress bar without any output in the terminal')
    parser.add_argument('-t', '--threads', type=int, help='Number of threads to use')
    parser.add_argument('-up', '--update', action='store_true', help='Update the tool automatically')
    parser.add_argument('-upt', '--updatetemplates', action='store_true', help='Update the templates')
    parser.add_argument('-h', '--help', action='store_true', help='Show help message and exit')

    args = parser.parse_args()

    if args.help:
        print_help()
    elif args.update:
        update_tool()
    elif args.updatetemplates:
        update_templates()
    else:
        if args.verbose:
            logging.getLogger().setLevel(logging.DEBUG)
        try:
            asyncio.run(check_version())
            asyncio.run(run_leakjs(urls_file=args.list, single_url=args.url, patterns_file=args.patterns, direct_patterns=args.regex, file_path=args.file, concurrency=args.concurrency, output_file=args.output, verbose=args.verbose, silent=args.silent))
        except re.error:
            print(f"[{Fore.RED} ERR {Style.RESET_ALL}] Something wrong with regex patterns")
if __name__ == "__main__":
    main()