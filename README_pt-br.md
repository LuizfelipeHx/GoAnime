<h4 align="center">
    <p>
        <b>Рortuguês</b> |
        <a href="https://github.com/alvarorichard/GoAnime/blob/main/README.md">English</a>
    </p>
</h4>

<p align="center">
  <img src="https://github.com/alvarorichard/GoAnime/assets/102667323/49600255-d5a2-4405-81d1-a08cebae569a" alt="Imagem logo" />
</p>

[![GitHub license](https://img.shields.io/github/license/alvarorichard/GoAnime)
](alvarorichard/GoAnime/blob/master/LICENSE)
![GitHub stars](https://img.shields.io/github/stars/alvarorichard/GoAnime)
![GitHub stars](https://img.shields.io/github/last-commit/alvarorichard/GoAnime)
![GitHub stars](https://img.shields.io/github/forks/alvarorichard/GoAnime?style=social)
[![Build Status](https://github.com/alvarorichard/GoAnime/actions/workflows/ci.yml/badge.svg)](https://github.com/alvarorichard/GoAnime/actions)
![GitHub contributors](https://img.shields.io/github/contributors/alvarorichard/GoAnime)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/9923765cb2854ae39af6b567996aad43)](https://app.codacy.com/gh/alvarorichard/GoAnime/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)
[![Build Status](https://app.travis-ci.com/alvarorichard/GoAnime.svg?branch=main)](https://app.travis-ci.com/alvarorichard/GoAnime)
[![Discord](https://img.shields.io/badge/Discord-Comunidade-7289DA?logo=discord&logoColor=white)](https://discord.gg/FbQuf78D9G)

# GoAnime

GoAnime é uma interface de usuário baseada em texto (TUI) simples, desenvolvida em Go, que permite aos usuários procurar animes e reproduzir ou baixar episódios diretamente no mpv. Ele coleta dados de sites para oferecer uma seleção de animes e episódios, com suporte a conteúdo legendado e dublado em inglês e português.

### Versão Mobile

Uma versão mobile do GoAnime está disponível para dispositivos Android: [GoAnime Mobile](https://github.com/alvarorichard/goanime-mobile)

> **Nota:** Esta versão está em desenvolvimento ativo e pode conter bugs ou funcionalidades incompletas.

### Comunidade


Entre no nosso Discord para suporte, feedback e novidades: [Servidor Discord](https://discord.gg/6nZ2SYv3)

## Recursos

- Buscar anime por nome
- Navegar pelos episódios
- Suporte a conteúdo legendado e dublado em inglês e português
- Pular introdução do anime
- Reproduzir online com seleção de qualidade
- Baixar episódios únicos
- Discord RPC sobre o anime
- Download em lote de múltiplos episódios
- Retomar reprodução de onde parou (em builds com suporte SQLite)
- Rastrear episódios assistidos (em builds com suporte SQLite)

> **Nota:** GoAnime pode ser compilado com ou sem suporte SQLite para rastreamento do progresso do anime.  
> [Veja a documentação das opções de build](docs/BUILD_OPTIONS.md) para mais detalhes.

> ⚠️ Aviso: disponibilidade da fonte em Português (PT-BR)
>
> Se você deseja assistir animes em português (PT-BR) e está fora do Brasil, será necessário usar uma VPN, proxy ou qualquer método para obter um endereço de IP brasileiro. A fonte de animes em PT-BR bloqueia o acesso de IPs fora do Brasil.

# Demo

<https://github.com/alvarorichard/GoAnime/assets/88117897/ffec6ad7-6ac1-464d-b048-c80082119836>

## Pré-requisitos

- Go (na versão mais recente)
- Mpv (na versão mais recente)

## Como instalar e executar

### Instalação Universal (Só precisa do go instalado e recomendado para a maioria dos usuários)

```shell
go install github.com/alvarorichard/Goanime/cmd/goanime@latest
```

### Métodos de instalação manual

```shell
git clone https://github.com/alvarorichard/GoAnime.git
```

```shell
cd GoAnime
```

```shell
go run cmd/goanime/main.go
```

## Filmes e Séries

GoAnime agora suporta filmes e séries através da fonte FlixHQ. Use a flag `--source flixhq` para buscar filmes e séries. Você também pode filtrar por tipo usando o parâmetro `--type` (por exemplo `--type movie` para buscar somente filmes).

```bash
# Buscar filmes/séries
goanime --source flixhq "The Matrix"

# Buscar somente filmes
goanime --source flixhq --type movie "Inception"

# Buscar somente séries
goanime --source flixhq --type tv "Breaking Bad"

# Habilitar legendas (inglês por padrão)
goanime --source flixhq --subs "Avatar"
```



## Linux

<details>
<summary>Arch Linux / Manjaro (sistemas baseados em AUR)</summary>

Usando Yay:

```bash
yay -S goanime
```

ou usando Paru:

```bash
paru -S goanime
```

Ou, para clonar e instalar manualmente:

```bash
git clone https://aur.archlinux.org/goanime.git
cd goanime
makepkg -si
sudo pacman -S mpv
```

</details>

<details>
<summary>Debian / Ubuntu (e derivados)</summary>

```bash
sudo apt update
sudo apt install mpv

# Para sistemas x86_64:
curl -Lo goanime https://github.com/alvarorichard/GoAnime/releases/latest/download/goanime-linux

chmod +x goanime
sudo mv goanime /usr/bin/
goanime
```

</details>

<details>
<summary>Instalação no Fedora</summary>

```bash
sudo dnf update
sudo dnf install mpv

# Para sistemas x86_64:
curl -Lo goanime https://github.com/alvarorichard/GoAnime/releases/latest/download/goanime-linux

chmod +x goanime
sudo mv goanime /usr/bin/
goanime
```

</details>

<details>
<summary>Instalação no openSUSE</summary>

```bash
sudo zypper refresh
sudo zypper install mpv

# Para sistemas x86_64:
curl -Lo goanime https://github.com/alvarorichard/GoAnime/releases/latest/download/goanime-linux

chmod +x goanime
sudo mv goanime /usr/bin/
goanime
```

</details>

## Windows

<details>
<summary>Instalação no Windows</summary>

> **Altamente Recomendado:** Use o instalador para a melhor experiência no Windows.

Opção 1: Usando o instalador (Recomendado)

- Baixe e execute o [Instalador do Windows](https://github.com/alvarorichard/GoAnime/releases/latest/download/GoAnimeInstaller.exe)

Opção 2: Executável independente

- Baixe o executável apropriado para seu sistema na [versão mais recente](https://github.com/alvarorichard/GoAnime/releases/latest)

</details>

## macOS

<details>
<summary>Instalação no macOS</summary>

Primeiro, instale o mpv usando o Homebrew:

```bash
# Instale o Homebrew se você ainda não tiver
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Instale o mpv
brew install mpv

# Baixe e instale o GoAnime
curl -Lo goanime https://github.com/alvarorichard/GoAnime/releases/latest/download/goanime-apple-darwin

chmod +x goanime
sudo mv goanime /usr/local/bin/
goanime
```

Instalação alternativa usando MacPorts:

```bash
# Instale o mpv usando MacPorts
sudo port install mpv

# Baixe e instale o GoAnime
curl -Lo goanime https://github.com/alvarorichard/GoAnime/releases/latest/download/goanime-apple-darwin

chmod +x goanime
sudo mv goanime /usr/local/bin/
goanime
```

</details>

### Passos de Configuração Adicionais

# Instalação no NixOS (Flakes)

## Execução Temporária

```shell
nix github:alvarorichard/GoAnime
```

## Instalação

Adicione no seu `flake.nix`:

```nix
 inputs.goanime.url = "github:alvarorichard/GoAnime";
```

Passe as entradas para seus módulos usando `specialArgs` e então no `configuration.nix`:

```nix
environment.systemPackages = [
  inputs.goanime.packages.${pkgs.system}.GoAnime
];
```

### Uso no Linux e macOS

```shell
go-anime
```

### Uso no Windows

```shell
goanime
```

### Uso Avançado

Você também pode usar parâmetros para procurar e reproduzir anime diretamente. Aqui estão alguns exemplos:

- Para procurar e reproduzir um anime diretamente, use o seguinte comando:

```shell
goanime  "nome do anime"
```

- Para atualizar o GoAnime para a versão mais recente, use a flag de atualização:

```shell
goanime --update
```

Este comando irá automaticamente baixar e instalar a versão mais recente do GoAnime usando o mecanismo de atualização integrado do Go.

Você pode usar a opção `-h` ou `--help` para exibir informações de ajuda sobre como usar o comando `goanime`.

```shell
goanime -h
```

O programa solicitará que você insira o nome de um anime. Digite o nome do anime que deseja assistir.

O programa apresentará uma lista de animes que correspondem à sua entrada. Navegue pela lista usando as setas do teclado e pressione enter para selecionar um anime.

Em seguida, o programa apresentará uma lista de episódios do anime selecionado. Novamente, navegue pela lista usando as setas do teclado e pressione enter para selecionar um episódio.

O episódio selecionado será então reproduzido no MPV.

# Agradecimentos

[@KitsuneSemCalda](https://github.com/KitsuneSemCalda), [@RushikeshGaikwad](https://github.com/Wraient) e [@the-eduardo](https://github.com/the-eduardo) por ajudar e melhorar essa aplicação.

# Alternativas

Se você estiver procurando por mais opções, confira este projeto alternativo do meu amigo [@KitsuneSemCalda](https://github.com/KitsuneSemCalda) chamado [Animatic-v2](https://github.com/KitsuneSemCalda/Animatic-v2), que foi inspirado no GoAnime.

## Contribuindo

Contribuições para melhorar ou aprimorar são sempre bem-vindas. Antes de contribuir, por favor leia nosso guia de desenvolvimento abrangente para informações detalhadas sobre nosso fluxo de trabalho, padrões de código e estrutura do projeto.

📖 **[Guia de Desenvolvimento](docs/Development.md)** - Leitura essencial para contribuidores

**Início Rápido para Contribuidores:**

1. Faça um fork do projeto
2. Leia o [Guia de Desenvolvimento](docs/Development.md) completamente
3. Crie sua branch de funcionalidade a partir de `dev` (nunca de `main`)
4. Siga nossos padrões de código (use `go fmt`, adicione comentários significativos)
5. Certifique-se de que todos os testes passem e adicione testes para novas funcionalidades
6. Faça commit das suas alterações usando formato de commit convencional
7. Faça push para sua branch
8. Abra um Pull Request para a branch `dev`

**Importante:** Nunca faça commit diretamente na branch `main`. Todas as mudanças devem passar pela branch `dev` primeiro.
