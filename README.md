<div align="center">
  <img src="https://github.com/user-attachments/assets/8c9b1877-5803-4c2a-a4ce-f22d5c1a2445" alt="ceye" width="500">
</div>


<br>
<br>
<br>


> [!NOTE] 
> **ceye is a minimal tool for monitoring certificate transparency logs without any hassle.**

<br>

- <sub> **detects new subdomains for your targets** </sub>
- <sub> **discord webhook notifications** </sub>
- <sub> **caches seen domains to avoid duplicates** </sub>

<br>
<br>

<h4>Installation</h4>

```bash
go install github.com/1hehaq/ceye@latest
```

<br>

<h6>setup configuration</h6>

```bash
# create config directory and provider template
ceye
```

- <sub>**edit the generated config file at `~/.config/ceye/provider.yaml`**</sub>

<div align="center">
  <img width="800" height="800" alt="image" src="https://github.com/user-attachments/assets/0e73cc08-46ec-4aa7-9da0-cf8e5fa0cf68" />
</div>

<br>
<br>

<h4>Flags</h4>

<pre>
  -target    : target domain to monitor (required if no config)
  -webhook   : discord webhook URL (required if no config)
  -version   : show version
  -update    : update to latest version
  -h, -help  : show help message
</pre>

<div align="center">
  <!-- <img alt="help" src="https://github.com/user-attachments/assets/957f6a89-08e2-4943-b0de-cf7296997ec6" /> -->
  <img alt="help" src="https://github.com/user-attachments/assets/a2302e5a-9024-48f7-935c-2e8465dd6aa0" />
</div>


<br>
<br>

<h4>Example Commands</h4>

```bash
# monitor github.com for new subdomains
ceye -target github.com -webhook https://discord.com/api/webhooks/...

# monitor multiple targets from config
ceye # config: ~/.config/ceye/provider.yaml
```

<br>

```bash
# get notified when companies add new subdomains
ceye -target tesla.com
```

<br>

```bash
# setup cron job to start ceye on system reboot
echo "@reboot ceye -target github.com -webhook YOUR_DISCORD_WEBHOOK" | crontab -
```

<br>
<br>

- **If you see no results or errors**
  - <sub> **verfiy your targets and webhook** </sub>
  - <sub> **check your internet connection** </sub>
  - <sub> **ensure docker is installed and running** </sub>
  - <sub> **use `-h` for guidance** </sub>

<br>
<br>

> [!CAUTION] 
> **never use `ceye` for any illegal activites, I'm not responsible for your deeds with it. Do for justice.**

<br>
<br>
<br>

<h6 align="center">kindly for hackers</h6>

<div align="center">
  <a href="https://github.com/1hehaq"><img src="https://img.icons8.com/material-outlined/20/808080/github.png" alt="GitHub"></a>
  <a href="https://twitter.com/1hehaq"><img src="https://img.icons8.com/material-outlined/20/808080/twitter.png" alt="X"></a>
</div></content>
