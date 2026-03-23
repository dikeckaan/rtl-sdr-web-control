# OpenWebRX on pi5

Bu klasor, `pi5.local` uzerinde RTL-SDR ile OpenWebRX calistirmak icin docker compose kurulumunu tutar.

## Dosyalar

- `docker-compose.yml`: OpenWebRX servisi
- `openwebrx/settings.json`: kalici receiver ve profil ayarlari

## Profiller

- `Broadcast FM`: `99.7 MHz`, `WFM`
- `2m`: `145.725 MHz`, `NFM`
- `70cm Repeaters`: `439.200 MHz`, `NFM`

## Calistirma

```bash
cd ~/sdr/openwebrx
docker compose up -d
```

Ardindan su adresi acin:

- `http://pi5.local:8073`
- veya `http://192.168.1.88:8073`

## Yonetim

```bash
cd ~/sdr/openwebrx
docker compose logs -f
docker compose restart
docker compose down
```

## Notlar

- Tek RTL-SDR ile ayni anda hem FM hem 70cm dinlenmez; profil degistirilir.
- `70cm` profili `439.200 MHz` icin hazirlandi.

## Hizli Komutlar

```bash
cd ~/sdr/openwebrx
./start.sh
./stop.sh
```

- `./start.sh`: OpenWebRX'i baslatir
- `./stop.sh`: compose servisini ve varsa eski `openwebrx` container'ini kapatir
