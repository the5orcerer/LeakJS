import subprocess
import sys
import os

def install_packages():
    """Install required packages from requirements.txt."""
    try:
        subprocess.check_call([sys.executable, "-m", "pip", "install", "-r", "requirements.txt", "--break"])
        print("[ INF ] Packages installed successfully.")
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while installing packages: {e}")
        sys.exit(1)

def move_and_setup():
    """Move and setup the main.py file."""
    try:
        os.rename("main.py", "leakjs")
        os.chmod("leakjs", 0o755)
        subprocess.check_call(["sudo", "mv", "leakjs", "/usr/local/bin"])
        print("[ INF ] Files moved and permissions set successfully.")
    except (OSError, subprocess.CalledProcessError) as e:
        print(f"[ ERR ] An error occurred while moving and setting up files: {e}")
        sys.exit(1)

def main():
    print("[ INF ] Starting installation process")
    install_packages()
    move_and_setup()
    print("[ INF ] Installation completed; Enjoy")

if __name__ == "__main__":
    main()
