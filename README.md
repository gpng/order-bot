# Order Telegram Bot

## Table of Contents

- [Usage](#usage)
- [Maintainers](#maintainers)
- [License](#license)

## Dependencies

[goose](https://github.com/pressly/goose)

[sqlc](https://dl.equinox.io/sqlc/sqlc/devel)

[swaggo](https://github.com/swaggo/swag)


## Usage

1. Create .env using .env.sample as example

1. Start docker containers

    ```
    make
    ```

1. View logs

    ```
    make logs
    ```

1. Visit `localhost:4000/` to check if API is responding

1. Generate docs from swagger comments

    ```
    make gen-docs
    ```

1. Visit `localhost:4000/docs` for documentation

1. Stop docker containers

    ```
    make down
    ```

## Migrations

Create new migrations

```
goose -dir sqlc/schemas create <migration_name> sql
```

Run migrations

```
env $(cat .env) make migrate
```

Rollback migrations

```
env $(cat .env) make rollback
```

## Generating models

```
make gen-models
```


## Test

```
make test
```

## Maintainers

[@gpng](https://github.com/gpng)

## License

MIT © 2020 Gerald Png
