## Instalacja zależności na systemie opartym na Debianie:

```sh
sudo apt install build-essential libusb-1.0-0-dev libfec-dev libfftw3-dev
```

## Kompilacja:

```sh
cd bin;
make clean;
make
```

## Sposób uruchomienia:

```sh
./gnssdec -{g/a/l} nagranie.bin
```