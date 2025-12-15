## Program do wykrywania spoofingu działający jako odrębny moduł wymaga jedynie jednej biblioteki:

```sh
pip install numpy
```

## Sposób uruchomienia:

```sh
./gnss_spoofing_server.py --system {g/a/l}
```

W celu wyświetlenia większej liczby informacji należy dodać flagę `--verbose`:

```sh
./gnss_spoofing_server.py --verbose --system {g/a/l}
```

Sama aplikacja uruchamia jedynie serwer. Program do dekodowania GNSS musi zostać uruchomiony osobno.