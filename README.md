# Stay Alive

The "Stay Alive" application is designed to keep your computer's session active by simulating human-like mouse movements. This tool is especially useful for preventing your screen from locking automatically when you're away from your keyboard for an extended period but need to keep your session active.

## Getting Started

Follow these instructions to get "Stay Alive" running on your machine for development and testing purposes.

### Prerequisites

Ensure you have Go installed on your machine. "Stay Alive" requires Go 1.24 or higher. You can check your Go version by running:

`go version`

If you need to install Go, follow the instructions on the [official Go website](https://golang.org/doc/install).

### Installing

Clone the repository to your local machine:

`git clone https://github.com/SlashGordon/stay-alive.git`

Navigate to the project directory:

`cd stay-alive`

Build the project:

`go build`

Run the application:

`./stay-alive`


## Usage

Stay Alive waits until you have not touched the mouse, trackpad or keyboard for 30 seconds. Then it moves the cursor a short way and back to roughly where it was. While you stay away it repeats that at random gaps of 15 to 30 seconds. As soon as you move the mouse or type, it stops and starts counting again. It never clicks or presses keys.

The moves follow a slightly curved path, speed up and slow down like a hand, shake a little, sometimes overshoot and correct, and rest briefly before heading back. Most moves are short nudges, a few go up to the full radius.

```
./stay-alive [flags]

  -idle duration       start moving the cursor after this much time without input (default 30s)
  -interval duration   longest pause between two moves while you stay idle (default 30s)
  -radius int          maximum distance of a move in pixels (default 100)
  -poll duration       how often to check for input (default 1s)
  -v                   log moves and detected activity
  -claude              exit once no Claude Code session is working
  -claude-quiet duration
                       a Claude session that wrote nothing for this long counts as done (default 15m)
```

Every flag can also be set with an environment variable. A flag on the command line wins over the variable.

| Variable             | Flag        | Example |
|----------------------|-------------|---------|
| `STAYALIVE_IDLE`     | `-idle`     | `2m`, `90s` or `90` (seconds) |
| `STAYALIVE_INTERVAL` | `-interval` | `45s` |
| `STAYALIVE_RADIUS`   | `-radius`   | `150` |
| `STAYALIVE_POLL`     | `-poll`     | `500ms` |
| `STAYALIVE_VERBOSE`  | `-v`        | `true` |
| `STAYALIVE_CLAUDE`   | `-claude`   | `true` |
| `STAYALIVE_CLAUDE_QUIET` | `-claude-quiet` | `30m` |

For example, in your `~/.zshrc`:

```
export STAYALIVE_IDLE=2m
export STAYALIVE_INTERVAL=45s
```

Pick an interval shorter than your screen lock timeout. Stop the program with Ctrl+C.

### Waiting for Claude Code

With `-claude` Stay Alive keeps the session awake only while Claude Code is working and exits once every session is done. It reads the transcripts Claude Code writes to `~/.claude/projects` (or `$CLAUDE_CONFIG_DIR/projects`) every 10 seconds and needs no hooks or other setup. Sessions that are already running when you start it are included.

A session counts as working while

- its last turn is open: a prompt waits for an answer, or a tool runs or has just returned,
- a background agent it started has not finished, or
- a workflow it started has not reached a final state.

A turn is over when Claude has answered, hit a usage limit or you interrupted it. Sessions that are open in a terminal or editor but wait for your next prompt do not count. Neither does a session that has written nothing for the `-claude-quiet` time, such as one waiting for a permission prompt or one that was closed mid-turn. When no session has been working for a minute, Stay Alive prints `stay-alive: all Claude Code sessions are done` and exits. The minute covers the gap between a background agent finishing and Claude picking up its result.

```
./stay-alive -claude -v
```

Background shell commands are not visible in the transcripts. If Claude ends its turn while such a command still runs, Stay Alive does not wait for it.

On macOS the terminal that runs Stay Alive needs the Accessibility permission (System Settings > Privacy & Security > Accessibility), otherwise the cursor does not move. Keyboard input is detected on macOS only. On Linux and Windows only mouse movement pauses the program.

## Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

Please refer to the [CONTRIBUTING.md](LINK_TO_CONTRIBUTING_GUIDELINES) for more information on how to submit pull requests.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/SlashGordon/stay-alive/tags).

## Authors

* **SlashGordon** - *Initial work* - [SlashGordon](https://github.com/SlashGordon)

See also the list of [contributors](https://github.com/SlashGordon/stay-alive/contributors) who participated in this project.