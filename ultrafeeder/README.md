# SDR Feed Stack

This directory contains the ADS-B feed stack for the RTL-SDR attached to `pi5.local`.

## Services

- `ultrafeeder`: reads the USB RTL-SDR, serves the local map, and provides Beast data
- `piaware`: feeds FlightAware from `ultrafeeder`
- `fr24feed`: feeds Flightradar24 from `ultrafeeder`

## Location

- Latitude: `40.94247`
- Longitude: `29.14824`
- Altitude: `82m` (`269ft`)

## IDs

- FlightAware feeder ID: `790d0a84-345b-4370-866d-501a467b9d93`
- Flightradar24 sharing key: `61109b8bce847030`
- Flightradar24 radar ID: `T-LTFJ69`

## Web UI

- Ultrafeeder map: `http://pi5.local:8080`
- Ultrafeeder map by IP: `http://192.168.1.88:8080`
- FR24 status page: `http://pi5.local:8754`

## Commands

Run these from this directory:

```sh
cd ~/sdr/ultrafeeder
```

Start or refresh containers:

```sh
docker compose up -d
```

Stop containers:

```sh
docker compose down
```

View status:

```sh
docker compose ps
```

Follow logs:

```sh
docker compose logs -f
```

Restart one service:

```sh
docker compose restart ultrafeeder
```

```sh
docker compose restart piaware
```

```sh
docker compose restart fr24feed
```

## Notes

- `ultrafeeder` is the only service that talks to the USB SDR directly.
- `piaware` and `fr24feed` both consume Beast data from `ultrafeeder`.
- MLAT is enabled for both FlightAware and FR24 in the current compose file.
- If FlightAware stats do not appear, verify the feeder is claimed in your FlightAware account.
- If FR24 shows feed problems, check `docker compose logs -f fr24feed` first.
