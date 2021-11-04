# Vault Backup

:warning: Check [the oficial way](https://learn.hashicorp.com/tutorials/vault/sop-backup) to backup your HashiCorp Vault.

Create a backup file of all HashiCorp Vault kv2 secrets.

```bash
./vault-backup -help
  -base64
        encode secret value as base64
  -filename string
        output filename (default "vault.backup")
  -help
        show this help output
  -output string
        output format. one of: json|yaml|kv (default "json")
  -paths string
        comma-separated base path. must end with /
```

Some environment variables must be defined before execution:

* `VAULT_TOKEN`: is required, retrieve one by running `vault login`;
* `VAULT_ADDR`: default value is `http://127.0.0.1:8200`

## Example

```bash
./vault-backup -base64 -filename my-vault.backup

2021/11/03 23:13:42 - reading production/app1/database
2021/11/03 23:13:42 - reading production/app1/cache
2021/11/03 23:13:43 - reading production/app2/database
2021/11/03 23:13:44 done! ;)
```

Backup file

```json
{
  "secret/data/production/app1/database/user": "dmF1bHQtYmFja3VwCg==",
  "secret/data/production/app1/database/password": "czNjcjN0Cg==",
  "secret/data/production/app1/cache/token": "czNjcjN0X19Ub2tlbgo=",
  "secret/data/production/app2/database/user": "dmF1bHQtYmFja3VwCg==",
  "secret/data/production/app2/database/password": "czNjcjN0Cg=="
}
```

