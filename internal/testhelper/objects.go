package testhelper

import "github.com/fsouza/fake-gcs-server/fakestorage"

var FakeObjects = []fakestorage.Object{
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music1.txt",
		},
		Content: []byte("some nice music"),
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music2.txt",
		},
		Content: []byte("some nicer music"),
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music3.txt",
		},
		Content: []byte("some even nicer music"),
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music4.mp3",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music5.wav",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music/1.txt",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music/2.txt",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music/3.txt",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/music/music/4.mp3",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "musics/musics/music1.txt",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "videos/video/video1_720p.mp4",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "videos/video/video1_480p.mp4",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "subs/video1.srt",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "your-bucket",
			Name:       "musics/music/music3.txt",
		},
		Content: []byte("wait what"),
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "videos/video/28043_1_video_1080p.mp4",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "videos/video/77071_1_caption_wg_240p_001f8ea7-749b-4d43-7bd5-b357e4e24f32.vtt",
		},
	},
	{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: "my-bucket",
			Name:       "videos/video/77071_1_caption_wg_240p_001f8ea7-749b-4d43-7bd5-b357e4e24f32.srt",
		},
	},
}
