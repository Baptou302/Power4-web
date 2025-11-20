import discord
from discord.ext import commands
import aiohttp
import asyncio
from aiohttp import web
import json
from datetime import datetime
from config import DISCORD_TOKEN, DISCORD_CHANNEL_ID, LOG_SERVER_HOST, LOG_SERVER_PORT

# Configuration du bot
intents = discord.Intents.default()
intents.message_content = True
bot = commands.Bot(command_prefix='!', intents=intents)

# Canal de logs
log_channel = None

@bot.event
async def on_ready():
    global log_channel
    print(f'Bot connecté en tant que {bot.user}')
    log_channel = bot.get_channel(DISCORD_CHANNEL_ID)
    if log_channel:
        print(f'Canal de logs configuré: {log_channel.name}')
    else:
        print(f'ERREUR: Canal avec l\'ID {DISCORD_CHANNEL_ID} introuvable!')
    
    # Démarrer le serveur HTTP pour recevoir les logs
    await start_log_server()

async def send_log(embed):
    """Envoie un embed dans le canal de logs"""
    if log_channel:
        try:
            await log_channel.send(embed=embed)
        except Exception as e:
            print(f"Erreur lors de l'envoi du log: {e}")

def create_embed(title, description, color, fields=None):
    """Crée un embed Discord"""
    embed = discord.Embed(
        title=title,
        description=description,
        color=color,
        timestamp=datetime.utcnow()
    )
    if fields:
        for field in fields:
            embed.add_field(
                name=field.get("name", ""),
                value=field.get("value", ""),
                inline=field.get("inline", False)
            )
    return embed

async def handle_log_event(request):
    """Gère les événements reçus depuis l'application Go"""
    try:
        data = await request.json()
        event_type = data.get("type")
        
        if event_type == "login":
            # Log de connexion
            username = data.get("username", "Inconnu")
            role = data.get("role", "user")
            role_emoji = "👑" if role == "admin" else "👤"
            
            embed = create_embed(
                title=f"{role_emoji} Connexion utilisateur",
                description=f"**{username}** s'est connecté",
                color=0x00ff00,  # Vert
                fields=[
                    {"name": "Type d'utilisateur", "value": role.upper(), "inline": True},
                    {"name": "Heure", "value": datetime.now().strftime("%H:%M:%S"), "inline": True}
                ]
            )
            await send_log(embed)
            
        elif event_type == "game_win":
            # Log de victoire
            username = data.get("username", "Inconnu")
            mode = data.get("mode", "Inconnu")
            difficulty = data.get("difficulty", "")
            is_ai_mode = data.get("is_ai_mode", False)
            
            mode_text = f"IA ({difficulty})" if is_ai_mode else "2 Joueurs"
            if difficulty:
                mode_text += f" - {difficulty}"
            
            embed = create_embed(
                title="🎉 Victoire",
                description=f"**{username}** a gagné une partie",
                color=0xffd700,  # Or
                fields=[
                    {"name": "Mode de jeu", "value": mode_text, "inline": True},
                    {"name": "Heure", "value": datetime.now().strftime("%H:%M:%S"), "inline": True}
                ]
            )
            await send_log(embed)
            
        elif event_type == "game_loss":
            # Log de défaite
            username = data.get("username", "Inconnu")
            mode = data.get("mode", "Inconnu")
            difficulty = data.get("difficulty", "")
            is_ai_mode = data.get("is_ai_mode", False)
            
            mode_text = f"IA ({difficulty})" if is_ai_mode else "2 Joueurs"
            if difficulty:
                mode_text += f" - {difficulty}"
            
            embed = create_embed(
                title="😔 Défaite",
                description=f"**{username}** a perdu une partie",
                color=0xff0000,  # Rouge
                fields=[
                    {"name": "Mode de jeu", "value": mode_text, "inline": True},
                    {"name": "Heure", "value": datetime.now().strftime("%H:%M:%S"), "inline": True}
                ]
            )
            await send_log(embed)
            
        elif event_type == "game_draw":
            # Log de match nul
            username = data.get("username", "Inconnu")
            mode = data.get("mode", "Inconnu")
            difficulty = data.get("difficulty", "")
            is_ai_mode = data.get("is_ai_mode", False)
            
            mode_text = f"IA ({difficulty})" if is_ai_mode else "2 Joueurs"
            if difficulty:
                mode_text += f" - {difficulty}"
            
            embed = create_embed(
                title="🤝 Match nul",
                description=f"**{username}** a fait un match nul",
                color=0x808080,  # Gris
                fields=[
                    {"name": "Mode de jeu", "value": mode_text, "inline": True},
                    {"name": "Heure", "value": datetime.now().strftime("%H:%M:%S"), "inline": True}
                ]
            )
            await send_log(embed)
            
        elif event_type == "game_start":
            # Log de début de partie
            username = data.get("username", "Inconnu")
            mode = data.get("mode", "Inconnu")
            difficulty = data.get("difficulty", "")
            is_ai_mode = data.get("is_ai_mode", False)
            
            mode_text = f"IA ({difficulty})" if is_ai_mode else "2 Joueurs"
            if difficulty:
                mode_text += f" - {difficulty}"
            
            embed = create_embed(
                title="🎮 Nouvelle partie",
                description=f"**{username}** a démarré une partie",
                color=0x0099ff,  # Bleu
                fields=[
                    {"name": "Mode de jeu", "value": mode_text, "inline": True},
                    {"name": "Heure", "value": datetime.now().strftime("%H:%M:%S"), "inline": True}
                ]
            )
            await send_log(embed)
        
        elif event_type == "xp_gain":
            # Log de gain d'XP
            username = data.get("username", "Inconnu")
            amount = data.get("amount", 0)
            old_xp = data.get("old_xp", 0)
            new_xp = data.get("new_xp", 0)
            old_level = data.get("old_level", 0)
            new_level = data.get("new_level", 0)
            level_up = data.get("level_up", False)
            
            # Déterminer le titre selon le niveau
            def get_title(level):
                if level >= 20:
                    return "Grand Maître"
                elif level >= 15:
                    return "Expérimenté"
                elif level >= 10:
                    return "Amateur"
                elif level >= 5:
                    return "Débutant"
                return "Novice"
            
            title = get_title(new_level)
            
            # Couleur selon si c'est un level up
            color = 0x9b59b6 if level_up else 0x3498db  # Violet si level up, bleu sinon
            
            # Description
            if level_up:
                description = f"**{username}** a gagné {amount} XP et est passé au niveau {new_level} ! 🎉"
            else:
                description = f"**{username}** a gagné {amount} XP"
            
            fields = [
                {"name": "XP gagné", "value": f"+{amount} XP", "inline": True},
                {"name": "XP total", "value": f"{old_xp} → {new_xp}", "inline": True},
                {"name": "Niveau", "value": f"{old_level} → {new_level}", "inline": True},
                {"name": "Titre", "value": title, "inline": True}
            ]
            
            if level_up:
                fields.append({"name": "🎊 Level Up !", "value": f"Niveau {old_level} → {new_level}", "inline": False})
            
            embed = create_embed(
                title="⭐ Gain d'XP" if not level_up else "🎊 Level Up !",
                description=description,
                color=color,
                fields=fields
            )
            await send_log(embed)
        
        return web.json_response({"status": "ok"})
        
    except Exception as e:
        print(f"Erreur lors du traitement de l'événement: {e}")
        return web.json_response({"status": "error", "message": str(e)}, status=500)

async def start_log_server():
    """Démarre le serveur HTTP pour recevoir les logs"""
    app = web.Application()
    app.router.add_post('/log', handle_log_event)
    
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, LOG_SERVER_HOST, LOG_SERVER_PORT)
    await site.start()
    print(f'Serveur de logs démarré sur http://{LOG_SERVER_HOST}:{LOG_SERVER_PORT}')

if __name__ == "__main__":
    bot.run(DISCORD_TOKEN)

