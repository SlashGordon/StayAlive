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
```

Every flag can also be set with an environment variable. A flag on the command line wins over the variable.

| Variable             | Flag        | Example |
|----------------------|-------------|---------|
| `STAYALIVE_IDLE`     | `-idle`     | `2m`, `90s` or `90` (seconds) |
| `STAYALIVE_INTERVAL` | `-interval` | `45s` |
| `STAYALIVE_RADIUS`   | `-radius`   | `150` |
| `STAYALIVE_POLL`     | `-poll`     | `500ms` |
| `STAYALIVE_VERBOSE`  | `-v`        | `true` |

For example, in your `~/.zshrc`:

```
export STAYALIVE_IDLE=2m
export STAYALIVE_INTERVAL=45s
```

Pick an interval shorter than your screen lock timeout. Stop the program with Ctrl+C.

On macOS the terminal that runs Stay Alive needs the Accessibility permission (System Settings > Privacy & Security > Accessibility), otherwise the cursor does not move. Keyboard input is detected on macOS only. On Linux and Windows only mouse movement pauses the program.

## Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

Please refer to the [CONTRIBUTING.md](LINK_TO_CONTRIBUTING_GUIDELINES) for more information on how to submit pull requests.

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/SlashGordon/stay-alive/tags).

## Authors

* **SlashGordon** - *Initial work* - [SlashGordon](https://github.com/SlashGordon)

See also the list of [contributors](https://github.com/SlashGordon/stay-alive/contributors) who participated in this project.