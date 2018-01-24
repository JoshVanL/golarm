package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/spf13/cobra"
)

const longForm = "January 2, 2006 at 3:04pm (MST)"
const FlagSoundFile = "file"

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringP(FlagSoundFile, "f", "~/sounds/alarm.mp3", "Set sound file path")
}

var RootCmd = &cobra.Command{
	Use:   "alarm",
	Short: "Simple alarm that takes as input a clock time hh:mm(am/pm) that will play an [mp3, flac, wav] sound file (e.g. 7:00am)",

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			ErrorHandler(errors.New("please supply one argument as the time for alarm (e.g. 7:00am)"))
		}

		soundFile, err := cmd.PersistentFlags().GetString(FlagSoundFile)
		if err != nil {
			ErrorHandler(fmt.Errorf("failed to parse file flag: %v", err))
		}

		path, err := ResolveFile(soundFile)
		if err != nil {
			ErrorHandler(err)
		}

		sound, err := DecodeSoundFile(path)
		if err != nil {
			ErrorHandler(err)
		}

		waitTime, err := GetWaitTime(args[0])
		if err != nil {
			ErrorHandler(err)
		}

		fmt.Printf("Using sound file: %v\n", soundFile)

		stopCh := SignalHandler(" Alarm cancelled.\n")

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			bar := New(waitTime.Seconds())
			ticker := time.NewTicker(time.Second)
			for bar.current < waitTime.Seconds()-1 {
				bar.Output()
				select {
				case <-ticker.C:
					bar.Increment()
				case <-stopCh:
					wg.Done()
					return
				}
			}
			bar.OutputDone()
			wg.Done()
		}()

		select {
		case <-time.Tick(waitTime):
			close(stopCh)
		case <-stopCh:
			wg.Wait()
			os.Exit(0)
		}

		wg.Wait()

		go func() {
			<-SignalHandler(" Alarm stopped.\n")
			os.Exit(0)
		}()

		for {
			fmt.Printf("Sounding Alarm!\n")

			done := make(chan struct{})

			speaker.Play(beep.Seq(sound, beep.Callback(func() {
				close(done)
			})))

			<-done

			sound, err = DecodeSoundFile(path)
			if err != nil {
				ErrorHandler(err)
			}

		}
	},
}

func ErrorHandler(err error) {
	fmt.Printf("error running alarm: %v\n", err)
	os.Exit(1)
}

func SignalHandler(message string) chan struct{} {
	stopCh := make(chan struct{})
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-ch:
			fmt.Print(message)
			close(stopCh)
		case <-stopCh:
		}
		signal.Stop(ch)
	}()

	return stopCh
}

func DecodeSoundFile(path string) (sound beep.StreamSeekCloser, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sound file: %v", err)
	}

	var s beep.StreamSeekCloser
	var format beep.Format

	switch filepath.Ext(path) {
	case ".mp3":
		s, format, err = mp3.Decode(f)
		break
	case ".flac":
		s, format, err = flac.Decode(f)
		break
	case ".wav":
		s, format, err = wav.Decode(f)
		break
	default:
		err = fmt.Errorf("only [mp3, flac, wav] supported: unable to use file '%s'", filepath.Base(path))
		break
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode sound file: %v", err)
	}
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	return s, nil
}

func ResolveFile(path string) (filePath string, err error) {
	file := path

	if path[0] == '~' {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}

		file = filepath.Join(usr.HomeDir, path[1:])
	}

	file, err = filepath.Abs(filepath.Dir(file))
	if err != nil {
		return "", fmt.Errorf("failed to resovle absolute path '%s': %v", path, err)
	}

	file = filepath.Join(file, filepath.Base(path))

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}

	return file, nil
}

func GetWaitTime(reqTime string) (time.Duration, error) {
	now := time.Now()
	strTime := fmt.Sprintf("%s %d, %d at %s (GMT)", now.Month().String(), now.Day(), now.Year(), reqTime)

	parsedTime, err := time.Parse(longForm, strTime)
	if err != nil {
		return 0, fmt.Errorf("failed to parse time: %v", err)
	}

	if time.Until(parsedTime) < 0 {
		parsedTime = parsedTime.Add(time.Hour * 24)
	}

	fmt.Printf("Setting alarm for: %s\n", parsedTime.String())

	return time.Until(parsedTime), nil
}
