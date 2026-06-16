Fork of https://github.com/BlackReloaded/wsl2-ssh-pageant
# wsl2-ssh-pageant

## Motivation
I use a Yubikey to store a GPG key pair and I like to use this key pair as my SSH key too. GPG on Windows exposes a Pageant style SSH agent and I wanted a way to use this key within WSL2.

## How to use with WSL2

### Prerequisite
In order to use `wsl-ssh-pageant` you must have installed `socat` and `ss` on your machine and unzip for the installation.

For example, on Ubuntu you can install these by running: `sudo apt install socat iproute2 unzip`

### Installation
1. Download latest version from [release page](https://github.com/jryberg/wsl2-ssh-pageant/releases/latest) and copy `wsl2-ssh-pageant.exe` to your windows home directory (or other location within the windows file system). Then simlink to your `$HOME/.ssh` directory for easy access
    ```bash
    windows_destination="/mnt/c/Users/Public/Downloads/wsl2-ssh-pageant.exe"
    linux_destination="$HOME/.ssh/wsl2-ssh-pageant.exe"
    wget "https://github.com/jryberg/wsl2-ssh-pageant/releases/latest/download/wsl2-ssh-pageant.exe.zip"
    unzip wsl2-ssh-pageant.exe.zip
    sudo mv wsl2-ssh-pageant.exe "$windows_destination"
    # Set the executable bit.
    sudo chmod +x "$windows_destination"
    # Symlink to linux for ease of use later
    ln -s $windows_destination $linux_destination
    ```
2. Add one of the following to your shell configuration (for e.g. `.bashrc`, `.zshrc` or `config.fish`). For advanced configurations consult the documentation of your shell.

#### Bash/Zsh

*SSH:*
```bash
ssh_agent_ok() {
  [ -S "$SSH_AUTH_SOCK" ] || return 1
  SSH_AUTH_SOCK="$SSH_AUTH_SOCK" ssh-add -l >/dev/null 2>&1
  [ $? -ne 2 ]   # ssh-add -l: 0=keys, 1=no keys but reachable, 2=cannot connect
}

if ! ssh_agent_ok; then
  wsl2_ssh_pageant_bin="$HOME/.ssh/wsl2-ssh-pageant.exe"
  if test -x "$wsl2_ssh_pageant_bin"; then
    # serialize concurrent shell startups so one rm -f can't unlink another's socket
    exec {lock_fd}>"$HOME/.ssh/.agent.lock"
    if flock -n "$lock_fd"; then
      if ! ssh_agent_ok; then                 # re-check inside the lock
        pkill -f "socat UNIX-LISTEN:$SSH_AUTH_SOCK" 2>/dev/null
        rm -f "$SSH_AUTH_SOCK"
        (setsid nohup socat \
           UNIX-LISTEN:"$SSH_AUTH_SOCK,fork,unlink-early" \
           EXEC:"$wsl2_ssh_pageant_bin" >/dev/null 2>&1 &)
        # brief verify: socat backgrounds, so wait for the socket then probe once
        for _ in 1 2 3 4 5; do [ -S "$SSH_AUTH_SOCK" ] && break; sleep 0.2; done
        ssh_agent_ok || echo >&2 "WARNING: ssh agent bridge (wsl2-ssh-pageant) failed to start."
      fi
    fi
    exec {lock_fd}>&-
    unset wsl2_ssh_pageant_bin
  else
    echo >&2 "WARNING: $wsl2_ssh_pageant_bin is not executable."
  fi
fi
unset -f ssh_agent_ok
```

*GPG:*
```bash
export GPG_AGENT_SOCK="$HOME/.gnupg/S.gpg-agent"
if ! ss -a | grep -q "$GPG_AGENT_SOCK"; then
  rm -rf "$GPG_AGENT_SOCK"
  wsl2_ssh_pageant_bin="$HOME/.ssh/wsl2-ssh-pageant.exe"
  if test -x "$wsl2_ssh_pageant_bin"; then
    (setsid nohup socat UNIX-LISTEN:"$GPG_AGENT_SOCK,fork" EXEC:"$wsl2_ssh_pageant_bin --gpg S.gpg-agent" >/dev/null 2>&1 &)
  else
    echo >&2 "WARNING: $wsl2_ssh_pageant_bin is not executable."
  fi
  unset wsl2_ssh_pageant_bin
fi
```

#### Fish

*SSH:*
```fish
set -x SSH_AUTH_SOCK "$HOME/.ssh/agent.sock"
if not ss -a | grep -q "$SSH_AUTH_SOCK";
  rm -f "$SSH_AUTH_SOCK"
  set wsl2_ssh_pageant_bin "$HOME/.ssh/wsl2-ssh-pageant.exe"
  if test -x "$wsl2_ssh_pageant_bin";
    setsid nohup socat UNIX-LISTEN:"$SSH_AUTH_SOCK,fork" EXEC:"$wsl2_ssh_pageant_bin" >/dev/null 2>&1 &
  else
    echo >&2 "WARNING: $wsl2_ssh_pageant_bin is not executable."
  end
  set --erase wsl2_ssh_pageant_bin
end
```

*GPG:*
```fish
set -x GPG_AGENT_SOCK "$HOME/.gnupg/S.gpg-agent"
if not ss -a | grep -q "$GPG_AGENT_SOCK";
  rm -rf "$GPG_AGENT_SOCK"
  set wsl2_ssh_pageant_bin "$HOME/.ssh/wsl2-ssh-pageant.exe"
  if test -x "$wsl2_ssh_pageant_bin";
    setsid nohup socat UNIX-LISTEN:"$GPG_AGENT_SOCK,fork" EXEC:"$wsl2_ssh_pageant_bin --gpg S.gpg-agent" >/dev/null 2>&1 &
  else
    echo >&2 "WARNING: $wsl2_ssh_pageant_bin is not executable."
  end
  set --erase wsl2_ssh_pageant_bin
end
```

## Troubleshooting

### Smartcard is detected in Windows and WSL, but ssh-add -L returns error
If this is the first time you using yubikey with windows with gpg4win, please follow the instructions in the link
https://developers.yubico.com/PGP/SSH_authentication/Windows.html

| Make sure ssh support is enabled in the `gpg-agent.conf` and restart `gpg-agent` with the following command

```
gpg-connect-agent killagent /bye
gpg-connect-agent /bye
```

### Agent response times are very slow
If ssh,ssh-add,gpg etc are very slow (~15-25 seconds) check that wsl2-ssh-pageant resides on the windows file system. This is due to an issue with the WSL interop documented [here](https://github.com/BlackReloaded/wsl2-ssh-pageant/issues/24) and [here](https://github.com/microsoft/WSL/issues/7591)

## Credit

Some of the code is copied from benpye's [wsl-ssh-pageant](https://github.com/benpye/wsl-ssh-pageant). This code shows how to communicate to pageant.
