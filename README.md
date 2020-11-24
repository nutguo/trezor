# trezord-go

官方 : https://github.com/trezor/trezord-go

```
$ ./trezord-go.exe  -h
Usage of C:\Users\Administrator\Desktop\trezord-go\trezord-go.exe:
  -db string
        Db path. Default, use 'trezord.db' in current directory
  -domains string
        Domains. Cors allow domains, split by ','
  -e value
        Use UDP port for emulator. Can be repeated for more ports. Example: trezord-go -e 21324 -e 21326
  -ed value
        Use UDP port for emulator with debug link. Can be repeated for more ports. Example: trezord-go -ed 21324:21326
  -l string
        Log into a file, rotating after 20MB
  -nocors
        Disable Cors check.
  -r    Reset USB device on session acquiring. Enabled by default (to prevent wrong device states); set to false if you plan to connect to debug link outside of bridge. (default true)
  -u    Use USB devices. Can be disabled for testing environments. Example: trezord-go -e 21324 -u=false (default true)
  -v    Write verbose logs to either stderr or logfile
  -version
        Write version


```