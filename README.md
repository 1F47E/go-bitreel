<div align="center">
<img src="https://github.com/kaspar1ndustries/go-bytereel/blob/master/docs/cover.png?raw=true" height="420">


<br><br>
<h1>bytereel converts any file to a video</h1>
<br><br>
</div>

### Yes but why?

So you can upload it to video hosting and have an infinite* tape storage like in a good old days.


### How it works?

Encoding file to a video is done by representing every bit as a black (1) or white (0) 2x2 pixels square.<br>
Due to this process, the resulting video will be approximately 4 times the size of your original file.<br>
A checksum for each frame is calculated and incorporated as metadata, ensuring the integrity of your data.<br>
The final step involves encoding these frames into a video using ffmpeg.<br>

<div align="center">
<br>
<a href="https://github.com/kaspar1ndustries/go-bytereel/blob/dev/docs/out.png?raw=true" target="_blank"><img src="https://github.com/kaspar1ndustries/go-bytereel/blob/dev/docs/out_cut.png?raw=true" width="300"></a>
</div>


### Dependencies

- ffmpeg

### Install
```
brew tap kaspar1ndustries/homebrew-tap
brew install bytereel
```


### TODO

- [x] encode file to frames
- [x] decode file from frames
- [x] save original filename
- [x] add metadata to every frame at the top
- [x] add checksum to frames at the end
- [x] detect end of the file
- [x] add bit rot protection - counting dominant pixels in a square
- [x] run ffmpeg from the code
- [x] generate video
- [x] decode video
- [x] check checksum on decode
- [x] add workers, limit to cpu cores
- [ ] release tap on homebrew
- [ ] add encryption
- [ ] error correction (ECC) like reed-solomon


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

Harder Drive: Hard drives we didn't want or need
https://www.youtube.com/watch?v=JcJSW7Rprio

Infinite Storage Glitch (rust)
https://github.com/DvorakDwarf/Infinite-Storage-Glitch


