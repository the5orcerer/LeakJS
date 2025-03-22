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
    """Move and setup the main.py file and handle .config/leakjs directory."""
    try:
        # Move main.py to leakjs and set permissions
        os.rename("main.py", "leakjs")
        os.chmod("leakjs", 0o755)
        subprocess.check_call(["sudo", "mv", "leakjs", "/usr/local/bin"])
        print("[ INF ] Files moved and permissions set successfully.")
        
        # Check if .config/leakjs directory exists, if not create it
        config_dir = os.path.expanduser("~/.config/leakjs")
        if not os.path.exists(config_dir):
            os.makedirs(config_dir)
            print(f"[ INF ] Directory {config_dir} created successfully.")
        
        # Move version.txt to .config/leakjs directory
        version_file = "version.txt"
        if os.path.exists(version_file):
            os.rename(version_file, os.path.join(config_dir, "version.txt"))
            print(f"[ INF ] {version_file} moved to {config_dir} successfully.")
        else:
            print(f"[ WRN ] {version_file} does not exist.")
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
