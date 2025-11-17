import os
from dotenv import load_dotenv
from pathlib import Path

# Charger le fichier .env depuis le répertoire du script
env_path = Path(__file__).parent / '.env'
if env_path.exists():
    # Lire le fichier manuellement pour gérer le BOM UTF-8
    try:
        with open(env_path, 'r', encoding='utf-8-sig') as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith('#') and '=' in line:
                    key, value = line.split('=', 1)
                    os.environ[key.strip()] = value.strip()
    except Exception as e:
        # Fallback vers load_dotenv standard
        load_dotenv(dotenv_path=env_path, override=True)
else:
    # Essayer aussi le répertoire courant
    load_dotenv(override=True)

# Configuration Discord
DISCORD_TOKEN = os.getenv("DISCORD_TOKEN", "").strip()
DISCORD_CHANNEL_ID_STR = os.getenv("DISCORD_CHANNEL_ID", "0").strip()

# Configuration du serveur HTTP pour recevoir les logs
LOG_SERVER_HOST = os.getenv("LOG_SERVER_HOST", "localhost").strip()
LOG_SERVER_PORT_STR = os.getenv("LOG_SERVER_PORT", "8080").strip()

# Conversion en entier
try:
    DISCORD_CHANNEL_ID = int(DISCORD_CHANNEL_ID_STR) if DISCORD_CHANNEL_ID_STR else 0
except ValueError:
    DISCORD_CHANNEL_ID = 0

try:
    LOG_SERVER_PORT = int(LOG_SERVER_PORT_STR) if LOG_SERVER_PORT_STR else 8080
except ValueError:
    LOG_SERVER_PORT = 8080

# Vérification des variables d'environnement
if not DISCORD_TOKEN:
    print(f"ERREUR: DISCORD_TOKEN non trouvé. Chemin .env: {env_path}")
    print(f"Fichier .env existe: {env_path.exists()}")
    if env_path.exists():
        try:
            print(f"Contenu du fichier .env:")
            with open(env_path, 'r', encoding='utf-8-sig') as f:
                content = f.read()
                print(content)
        except Exception as e:
            print(f"Erreur lors de la lecture: {e}")
    raise ValueError("DISCORD_TOKEN n'est pas défini dans le fichier .env")
if DISCORD_CHANNEL_ID == 0:
    raise ValueError("DISCORD_CHANNEL_ID n'est pas défini dans le fichier .env")

