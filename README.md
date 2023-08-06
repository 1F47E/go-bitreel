![bytereel - converts any file to a video](assets/banner.png)

<div align="center">
<br><br>
<h1>bytereel</h1>
<h3>Convert any file to a video</h3>
<br><br>
</div>

### Yes but why?

So you can upload it a to video hosting and have an infinite* tape storage.


### How does it work?

Encoding file to a video is done by representing every bit as a black (1) or white (0) 2x2 pixels square.<br>
Due to this process, the resulting video will be approximately 4 times the size of your original file.<br>
A checksum for each frame is calculated and incorporated as metadata, ensuring the integrity of your data.<br>
The final step involves encoding these frames into a video using ffmpeg.<br>

<div align="center">
<br>
<img src="assets/screenshot.png" width="600"></a>
<br>
<br>
<br>
<a href="assets/out.png" target="_blank">
<img src="assets/out_cut.png" width="420"></a>
<br>
</div>



### Dependencies

```
brew install ffmpeg
```

### Install
```
brew tap 1F47E/homebrew-tap
brew install bytereel
```

### Usage

To encode a file
```
bytereel encode <file>
```

To decode a file
```
bytereel decode <file>
```


### DEV NOTES
encode images to video with image convert to yuv422p10
```
ffmpeg -framerate 30 -i out_%d.png -c:v prores -profile:v 3 -pix_fmt yuv422p10 output.mov
```

decode video to images
```
ffmpeg -i tmp/out/output.mov tmp/out/decompressed/output_%08d.png
```

### Inspiration

Infinite Storage Glitch (rust)
https://github.com/DvorakDwarf/Infinite-Storage-Glitch


