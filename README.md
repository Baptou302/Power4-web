# 🎮 Power4 Web

<div align="center">

**Un jeu de Puissance 4 moderne et interactif développé en Go avec une interface web élégante**

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

*Projet réalisé par **Baptiste Lecoq** - Étudiant en B1 Informatique*

</div>

---

## 📋 Table des matières

- [À propos](#-à-propos)
- [Fonctionnalités](#-fonctionnalités)
- [Technologies utilisées](#-technologies-utilisées)
- [Prérequis](#-prérequis)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Utilisation](#-utilisation)
- [Structure du projet](#-structure-du-projet)
- [Captures d'écran](#-captures-décran)
- [Auteur](#-auteur)

---

## 🎯 À propos

**Power4 Web** est une implémentation moderne du célèbre jeu de Puissance 4 (Connect Four) accessible via un navigateur web. Le projet combine une architecture backend robuste en Go avec une interface utilisateur moderne et intuitive, offrant une expérience de jeu fluide et sécurisée.

### Caractéristiques principales

- 🎲 **Modes de jeu variés** : Joueur contre joueur, contre IA avec différents niveaux de difficulté
- 🔐 **Système d'authentification** : Gestion des utilisateurs avec sessions sécurisées
- 🎵 **Lecteur audio intégré** : Ambiance musicale pendant les parties
- 🤖 **IA avancée** : Intelligence artificielle avec plusieurs niveaux de difficulté
- 📊 **Tableau de bord administrateur** : Gestion des utilisateurs et statistiques
- 🔒 **Sécurité HTTPS** : Communication chiffrée avec certificats auto-signés
- 📝 **Système de logs Discord** : Intégration avec un bot Discord pour le suivi des activités

---

## ✨ Fonctionnalités

### 🎮 Modes de jeu

- **Mode Local** : Deux joueurs sur le même appareil
- **Mode IA Facile** : Défiez une intelligence artificielle de niveau débutant
- **Mode IA Moyen** : Affrontez une IA de niveau intermédiaire
- **Mode IA Difficile** : Testez vos compétences contre une IA experte

### 🔐 Authentification

- Inscription et connexion sécurisées
- Gestion de sessions avec cookies sécurisés
- Protection des routes sensibles via middleware

### 🎵 Lecteur audio

- Lecteur audio intégré avec contrôles complets
- Playlist de musique intégrée
- Interface élégante et non-intrusive

### 📊 Administration

- Tableau de bord administrateur
- Visualisation des utilisateurs
- Statistiques de jeu

---

## 🛠️ Technologies utilisées

### Backend
- **Go 1.25** : Langage principal pour le serveur
- **MySQL** : Base de données pour la persistance
- **Gorilla Sessions** : Gestion des sessions utilisateur
- **HTTPS/TLS** : Communication sécurisée

### Frontend
- **HTML5** : Structure des pages
- **CSS3** : Styles modernes avec animations
- **JavaScript** : Interactivité et logique client

### Outils
- **Python** : Bot Discord pour les logs
- **mkcert** : Génération de certificats SSL
- **Discord.py** : API Discord pour le bot

---

## 📦 Prérequis

Avant de commencer, assurez-vous d'avoir installé :

- [Go](https://golang.org/dl/) version 1.25 ou supérieure
- [MySQL](https://dev.mysql.com/downloads/) 8.0 ou supérieure
- [Python](https://www.python.org/downloads/) 3.8 ou supérieure (pour le bot Discord)
- Un navigateur web moderne (Chrome, Firefox, Edge, Safari)

### Optionnel
- Un serveur Discord et un bot Discord (pour les logs)

---

## 🚀 Installation

### 1. Cloner le dépôt

```bash
git clone https://github.com/Baptou302/Power4-web.git
cd Power4-web
```

### 2. Installer les dépendances Go

```bash
go mod download
```

### 3. Configurer la base de données

Créez une base de données MySQL :

```sql
CREATE DATABASE power4_db;
```

### 4. Configurer les variables d'environnement

Créez un fichier `.env` à la racine du projet (ou configurez les variables système) :

```env
DB_USER=votre_utilisateur_mysql
DB_PASSWORD=votre_mot_de_passe
DB_NAME=power4_db
DB_HOST=localhost
DB_PORT=3306
HTTPS_PORT=3443
```

### 5. Configurer le bot Discord (optionnel)

Si vous souhaitez utiliser le bot Discord pour les logs :

1. Créez un bot sur le [Discord Developer Portal](https://discord.com/developers/applications)
2. Copiez le token du bot
3. Créez un fichier `bot/config.py` :

```python
DISCORD_TOKEN = "votre_token_bot"
DISCORD_CHANNEL_ID = votre_id_canal
LOG_SERVER_HOST = "localhost"
LOG_SERVER_PORT = 8080
```

---

## ⚙️ Configuration

### Variables d'environnement

Le projet utilise les variables d'environnement suivantes :

| Variable | Description | Valeur par défaut |
|----------|-------------|-------------------|
| `DB_USER` | Utilisateur MySQL | `root` |
| `DB_PASSWORD` | Mot de passe MySQL | (requis) |
| `DB_NAME` | Nom de la base de données | `power4_db` |
| `DB_HOST` | Hôte MySQL | `localhost` |
| `DB_PORT` | Port MySQL | `3306` |
| `HTTPS_PORT` | Port HTTPS du serveur | `3443` |

---

## 🎮 Utilisation

### Démarrer le serveur

```bash
go run main.go
```

Le serveur démarrera sur `https://localhost:3443`

> **Note** : Les certificats SSL auto-signés seront générés automatiquement lors du premier lancement. Votre navigateur vous demandera d'accepter le certificat.

### Démarrer le bot Discord (optionnel)

Dans un terminal séparé :

```bash
cd bot
python bot.py
```

### Accéder à l'application

1. Ouvrez votre navigateur
2. Naviguez vers `https://localhost:3443`
3. Acceptez le certificat SSL (auto-signé)
4. Créez un compte ou connectez-vous
5. Choisissez votre mode de jeu et amusez-vous !

---

## 📁 Structure du projet

```
Power4-web/
│
├── assets/                 # Ressources statiques
│   ├── music/             # Fichiers audio
│   ├── musique/           # Images des albums
│   └── style/             # Feuilles de style CSS
│
├── bot/                   # Bot Discord pour les logs
│   ├── bot.py            # Code principal du bot
│   └── config.py         # Configuration du bot
│
├── docs/                  # Documentation du projet
│   └── documentation projet/
│
├── src/                   # Code source Go
│   ├── auth/             # Authentification et sessions
│   ├── game/             # Logique du jeu et IA
│   ├── handlers/         # Gestionnaires HTTP
│   ├── logger/           # Système de logging
│   ├── middleware/       # Middleware HTTP
│   ├── models/           # Modèles de données et DB
│   ├── server/           # Configuration du serveur
│   └── utils/            # Utilitaires (certificats, etc.)
│
├── templates/             # Templates HTML
│   ├── admin/            # Pages administrateur
│   ├── auth/             # Pages d'authentification
│   └── index/            # Pages principales
│
├── main.go               # Point d'entrée de l'application
├── go.mod               # Dépendances Go
├── go.sum               # Checksums des dépendances
└── README.md            # Ce fichier
```

---
## 👤 Auteur

**Baptiste Lecoq**

- Étudiant en **B1 Informatique**
---

<div align="center">

**Fait avec ❤️ par Baptiste Lecoq**

*Bon jeu ! 🎮*

</div>
