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
import json

init(autoreset=True)

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

def banner():
    init(autoreset=True)

    print(f"""
                        __    v1.0       __       _______
                       / /   ___  ____ _/ /__    / / ___/
                      / /   / _ \\/ __ `/ //_/_  / /\\__ \\ 
                     / /___/  __/ /_/ / ,< / /_/ /___/ / 
                    /_____/\\___/\\__,_/_/|_|\\____//____/  
                                               {Style.BRIGHT}{Fore.RED}@rootplinix
            """)

def read_urls(file_path: str) -> List[str]:
    """Read URLs from a file and return a sorted list of .js URLs."""
    with open(file_path, 'r') as file:
        urls = [line.strip() for line in file if line.strip().endswith('.js')]
    return sorted(urls)

def read_patterns(yaml_file: str) -> List[Dict[str, Any]]:
    """Read patterns from a YAML file."""
    with open(yaml_file, 'r') as file:
        data = yaml.safe_load(file)
    return data['patterns']

def load_default_patterns() -> List[Dict[str, Any]]:
    """Load default patterns from the .config/LeakJS directory."""
    patterns = []
    config_dir = os.path.expanduser("~/.config/LeakJS")
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
            patterns.append({
                'pattern': {
                    'name': 'DirectPattern',
                    'regex': pattern.strip(),
                    'confidence': 'N/A'
                }
            })
    return patterns

def read_file_content(file_path: str) -> str:
    """Read the content of a file."""
    try:
        with open(file_path, 'r') as file:
            return file.read()
    except FileNotFoundError:
        logging.error(f"File not found: {file_path}")
        return ""
    except IOError as e:
        logging.error(f"Error reading file {file_path}: {e}")
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

def display_results(source: str, results: Dict[str, Dict[str, Any]], source_type: str = "URL") -> None:
    """Display the results of pattern matching."""
    print(f"{Fore.CYAN}[ * ] {source} ({source_type})\n")
    for name, data in results.items():
        print(f"{Fore.YELLOW}[{name}]")
        for match in data['matches']:
            print(f"{Fore.GREEN}[{match}]")
        print(f"{Fore.MAGENTA}[Confidence: {data['confidence']}]\n")

def save_results_to_file(output_file: str, source: str, results: Dict[str, Dict[str, Any]], source_type: str = "URL") -> None:
    """Save the results of pattern matching to a file."""
    with open(output_file, 'a') as file:
        file.write(f"[ * ] {source} ({source_type})\n")
        for name, data in results.items():
            file.write(f"[{name}]\n")
            for match in data['matches']:
                file.write(f"[{match}]\n")
            file.write(f"[Confidence: {data['confidence']}]\n")
        file.write("\n")

async def fetch_url_content(url: str, session: ClientSession) -> str:
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
        logging.error(f"Error fetching URL {url}: {e}")
        return ""

async def process_url(url: str, patterns: List[Dict[str, Any]], session: ClientSession, output_file: Optional[str] = None, verbose: bool = False) -> None:
    """Process a single URL to fetch content and search for patterns."""
    content = await fetch_url_content(url, session)
    if content:
        results = search_patterns(content, patterns)
        if results:
            if output_file:
                save_results_to_file(output_file, url, results)
            else:
                display_results(url, results)
        else:
            if verbose:
                logging.info(f"No matches found in {url}")

async def process_file(file_path: str, patterns: List[Dict[str, Any]], output_file: Optional[str] = None, verbose: bool = False) -> None:
    """Process a single file to read content and search for patterns."""
    content = read_file_content(file_path)
    if content:
        results = search_patterns(content, patterns)
        if results:
            if output_file:
                save_results_to_file(output_file, file_path, results, source_type="File")
            else:
                display_results(file_path, results, source_type="File")
        else:
            if verbose:
                logging.info(f"No matches found in {file_path}")

async def worker(queue: Queue, patterns: List[Dict[str, Any]], source_type: str, output_file: Optional[str] = None, verbose: bool = False) -> None:
    """Worker function for threading."""
    async with RetryClient(raise_for_status=False) as session:
        while not queue.empty():
            source = queue.get()
            if source_type == "URL":
                await process_url(source, patterns, session, output_file, verbose)
            elif source_type == "File":
                await process_file(source, patterns, output_file, verbose)
            queue.task_done()

async def run_leakjs(urls_file: Optional[str], single_url: Optional[str], patterns_file: Optional[str], direct_patterns: Optional[str], file_path: Optional[str], concurrency: int, output_file: Optional[str], verbose: bool) -> None:
    """Main function to handle input and start the scanning process."""
    patterns = load_default_patterns()
    if patterns_file:
        patterns.extend(read_patterns(patterns_file))
    if direct_patterns:
        patterns.extend(parse_direct_patterns(direct_patterns))

    urls = []
    if urls_file:
        urls = read_urls(urls_file)
    elif single_url:
        urls = [single_url]
    elif not sys.stdin.isatty():
        urls = [line.strip() for line in sys.stdin if line.strip().endswith('.js')]

    if not urls and not file_path:
        logging.error("No URLs or files provided.")
        return

    queue = Queue()
    source_type = "URL"
    if urls:
        for url in urls:
            queue.put(url)
    if file_path:
        queue.put(file_path)
        source_type = "File"

    tasks = []
    for _ in range(concurrency):
        task = asyncio.create_task(worker(queue, patterns, source_type, output_file, verbose))
        tasks.append(task)

    await asyncio.gather(*tasks)

def print_help() -> None:
    """Print help message."""
    help_text = """
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
      -h, --help        Show this help message and exit
    """
    print(help_text)

def handle_sigint(signal, frame):
    print(f"\n{Fore.RED}Process interrupted. Exiting gracefully...{Style.RESET_ALL}")
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
    parser.add_argument('-h', '--help', action='store_true', help='Show help message and exit')

    args = parser.parse_args()

    if args.help:
        print_help()
    else:
        if args.verbose:
            logging.getLogger().setLevel(logging.DEBUG)
        asyncio.run(run_leakjs(urls_file=args.list, single_url=args.url, patterns_file=args.patterns, direct_patterns=args.regex, file_path=args.file, concurrency=args.concurrency, output_file=args.output, verbose=args.verbose))

if __name__ == "__main__":
    main()