# Twitter-Clone.go

**Este projeto eu fiz para aprender como fazer um servidor HTTP. É uma simulação do twitter do passarinho azul. Com ele aprendi sobre:**
- APIs REST, como fazer endpoints usando verbos apropriados. 
- Autenticação e segurança incluindo hashing de senhas, o uso de JWT e proteção de rotas.
- O uso do PostgresSQL, como fazer migrações e gerenciar dados.
- Por fim a implementação de webhooks e a integração segura com sistemas externos.
- Além de demonstrar o uso dos conceitos básicos de Go(~~Se até o typescript se rendeu eu também me rendo~~)

## Features

- **Autenticação de Usuário**
  - Registro com e-mail/senha
  - Autenticação baseada em JWT com tokens de atualização
  - Atualizações de conta (e-mail, senha)
  - Gerenciamento de sessão (login, logout, tokens de acesso)

- **Chirps (Tweets)**
  - Criar chirps (máximo de 140 caracteres)
  - Recuperar todos os chirps com ordenação opcional
  - Filtrar chirps por autor
  - Excluir seus próprios chirps
  - Filtragem automática de palavrões

- **Recursos Premium**
  - Suporte à assinatura Chirpy Red via integração com Polka(webhook ficticio)

- **Funções de Administrador**
  - Visualizar métricas de uso do servidor
  - Resetar usuários (apenas no modo de desenvolvimento)

## Primeiros Passos

### Pré-requisitos

- Go 1.22 ou superior
- PostgreSQL
- Git

### Instalação

1. Clone o repositório
   ```
   git clone https://github.com/PedroMartini98/Twitter-Clone.go.git
   cd Twitter-Clone.go
   ```

2. Configure as variáveis de ambiente (crie um arquivo `.env`)
   ```
   DB_URL=postgresql://username:password@localhost:5432/chirpy?sslmode=disable
   JWT_SECRET=sua_chave_secreta_jwt
   POLKA_KEY=sua_chave_de_integracao_polka
   PLATFORM=dev  # Use 'prod' para produção
   ```

3. Instale as dependências
   ```
   go mod download
   ```

4. Execute o servidor
   ```
   go run main.go
   ```

O servidor iniciará na porta 8080.

## API Endpoints

### Health Check
- `GET /api/healthz` - Verifica se a servidor tá funcionando

### Users
- `POST /api/users` - Cria um novo usuário
- `PUT /api/users` - Modifica os dados de um usuário (requer autenticação)

### Authentication
- `POST /api/login` - Login com email e senha
- `POST /api/refresh` - Pega um novo token de acesso usando um refresh token
- `POST /api/revoke` - Revoga um refresh token manualmente (logout)

### Chirps
- `POST /api/chirps` - Cria um novo chirp (requer autenticação)
- `GET /api/chirps` - Recebe todos os chirps
  - Parametros de busca:
    - `author_id` - Filtar por usuário
    - `sort` - Ordena os chirps por ordem de criação (`asc` or `desc`)
- `GET /api/chirps/{chirpId}` - Pega um chirp espicífo pelo id
- `DELETE /api/chirps/{chirpId}` - Excluir um chirp (requer autenticação do criador do chirp)

### Polka Integration
- `POST /api/polka/webhooks` - Endpoint do webhook do "Polka" (requer chave de API)

### Admin
- `GET /admin/metrics` - Visualizar o uso do servidor
- `POST /admin/reset` - Reseta os usuários (mais pra função de testes)

## Exemplos de Request/Response

### Create a User
```
POST /api/users
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

Response:
```json
{
  "id": "a1b2c3d4-e5f6-7890-a1b2-c3d4e5f67890",
  "created_at": "2023-07-31T12:34:56Z",
  "updated_at": "2023-07-31T12:34:56Z",
  "email": "user@example.com",
  "is_chirpy_red": false
}
```

### Login
```
POST /api/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}
```

Response:
```json
{
  "id": "a1b2c3d4-e5f6-7890-a1b2-c3d4e5f67890",
  "created_at": "2023-07-31T12:34:56Z",
  "updated_at": "2023-07-31T12:34:56Z",
  "email": "user@example.com",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "abc123def456ghi789jkl",
  "is_chirpy_red": false
}
```

### Create a Chirp
```
POST /api/chirps
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json

{
  "body": "Hello world! This is my first chirp."
}
```

Response:
```json
{
  "id": "b1c2d3e4-f5g6-7890-b1c2-d3e4f5g67890",
  "created_at": "2023-07-31T12:40:56Z",
  "updated_at": "2023-07-31T12:40:56Z",
  "body": "Hello world! This is my first chirp.",
  "user_id": "a1b2c3d4-e5f6-7890-a1b2-c3d4e5f67890"
}
```

## Segurança

- Senhas são criptografadas antes do armazenamento
- Tokens JWT expiram após 1 hora
- Refresh tokens podem ser revogados
- Chaves de API são necessárias para integração do webhook

## Desenvolvimento

### Geração de Código SQL

Este projeto utiliza SQLc para gerar código Go a partir de consultas SQL. Para regenerar o código após alterações nas consultas:

```
sqlc generate
```

### Resetar Dados (Apenas em Desenvolvimento)

Para resetar todos os usuários no modo de desenvolvimento:
```
POST /admin/reset
```

### Estrutura de Arquivos
```
/
├── main.go                    # Arquivo principal da aplicação
├── assets/                    # Ativos estáticos
├── internal/                  # Pacotes internos
│   ├── auth/                  # Lógica de autenticação
│   └── database/              # Código gerado para o banco de dados
├── sql/                       # Arquivos SQL
│   ├── schema/                # Esquemas de migração
│   │   ├── 001_users.sql      # Criação da tabela de usuários
│   │   ├── 002_chirps.sql     # Criação da tabela de chirps
│   │   ├── 003_passwrods.sql  # Alterações na tabela de senhas
│   │   ├── 004_*.sql          # Scripts de migração para tokens
│   │   └── 005_*.up.sql       # Script para adicionar Chirpy Red
│   └── queries/               # Consultas SQL
│       └── queries.sql        # Consultas para geração de código
├── sqlc.yaml                  # Configuração do SQLc
├── go.mod                     # Dependências Go
└── go.sum                     # Checksums das dependências
```
