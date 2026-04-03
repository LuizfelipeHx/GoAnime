# GoAnime Desktop (Wails + React)

Este diretório contém a versão desktop do GoAnime usando Wails (backend Go + frontend React).

## Rodar em modo desenvolvimento

### Opção mais fácil (Windows)

Clique duas vezes em:

- `desktop/run-desktop.bat`

ou execute no PowerShell:

- `./desktop/run-desktop.ps1`

### Manual

1. Instale dependências do frontend:

```powershell
cd desktop/frontend
npm.cmd install
npm.cmd run build
```

2. Volte para a pasta desktop e inicie o app:

```powershell
cd ..
# se já tiver o CLI wails
wails dev

# fallback sem instalar CLI global
# go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev
```

## Build desktop

```powershell
cd desktop
wails build
```

## O que já está pronto

- Busca multi-fonte (AllAnime, AnimeFire, FlixHQ, AnimesOnlineCC)
- UI moderna em React
- Listagem de episódios
- Player integrado
- Proxy interno para stream HLS (sem usuário precisar abrir localhost manual)

## Limitações atuais

- Fluxo de TV do FlixHQ ainda parcial no desktop (temporada/episódio)
- Alguns hosts podem bloquear stream por regras do provedor
