# Single shared asset root on NetStorage; **brand subfolders** selected by hostname.

| Hostname | Brand folder | Example URL | NetStorage path |
|----------|--------------|-------------|-----------------|
| `assets.cubcadet.com` | `cubcadet` | `/na/images/hero.jpeg` | `/{cpcode}/assets/cubcadet/na/images/hero.jpeg` |
| `assets.genuinefactoryparts.com` | `gfp` | `/na/images/hero.jpeg` | `/{cpcode}/assets/gfp/na/images/hero.jpeg` |

Mapping is in `snippets/brand-folder-map.conf` (`map $host $ns_brand_folder`).

## Folder layout

```
static/assets/
  cubcadet/
    na/
      images/
        hero.jpeg
  gfp/
    na/
      images/
        hero.jpeg
```

NetStorage: upload the same tree under `/{cpcode}/assets/`.

## Local testing

Hosts:

```
127.0.0.1 assets.cubcadet.com assets.genuinefactoryparts.com
```

```bash
curl http://assets.cubcadet.com:8080/na/images/hero.jpeg      # cubcadet folder
curl http://assets.genuinefactoryparts.com:8080/na/images/hero.jpeg  # gfp folder
```

## Add a brand

1. Add folder `static/assets/newbrand/…` and upload to `/{cpcode}/assets/newbrand/…`.
2. Add hostname → folder in `snippets/brand-folder-map.conf` (both `map` blocks).
3. Add `assets.newbrand.com` to `server_name` in `conf.d/domains*.conf`.

## Both brands on one hostname (optional)

If you need **one** domain with path prefixes instead of separate asset hosts:

```nginx
location /cubcadet/ {
    alias /var/www/assets/cubcadet/;
}
location /gfp/ {
    alias /var/www/assets/gfp/;
}
```

URLs become `assets.cubcadet.com/cubcadet/na/images/hero.jpeg`. The default setup uses **hostname** so URLs stay `assets.cubcadet.com/na/...` without a brand prefix.
