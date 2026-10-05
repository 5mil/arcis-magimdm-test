# MagiMDM for Windows

One file: `magimdm.exe`. Parent desk for a small fleet. Devices, policy, enroll, and a link to Arcis on this PC.

Arcis stays the library (`arcis.exe` on `127.0.0.1:9090`). MagiMDM does not enroll students into that console.

```bat
magimdm.exe
magimdm.exe --self-test
```

Default parent login is `admin` / `changeme`. Change it after first open.

Windows agents enroll at `http://127.0.0.1:8788/api/agent/enroll`, the same path as `agent-pc/windows/enroll.ps1`.
