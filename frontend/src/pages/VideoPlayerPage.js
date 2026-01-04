import React, { useEffect, useState, useRef } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import {
  Container,
  Box,
  Typography,
  IconButton,
  AppBar,
  Toolbar,
  Paper,
  CircularProgress,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  CardMedia,
  List,
  ListItem,
  ListItemButton,
  Divider,
} from '@mui/material';
import {
  ArrowBack,
  ThumbUp,
  ThumbUpOutlined,
  Fullscreen,
  FullscreenExit,
} from '@mui/icons-material';
import {
  getVideo,
  getVideoStreamUrl,
  incrementView,
  toggleLike,
  getComments,
  addComment,
  getPlaylist,
  getVideos,
} from '../api';
import CommentSection from '../components/CommentSection';

const VideoPlayerPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const videoRef = useRef(null);
  const containerRef = useRef(null);
  const [video, setVideo] = useState(null);
  const [loading, setLoading] = useState(true);
  const [liked, setLiked] = useState(false);
  const [localLikes, setLocalLikes] = useState(0);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const [comments, setComments] = useState([]);
  const [playlist, setPlaylist] = useState(null);
  const [playlistVideos, setPlaylistVideos] = useState([]);
  const [currentVideoIndex, setCurrentVideoIndex] = useState(0);

  useEffect(() => {
    const fetchVideo = async () => {
      try {
        const data = await getVideo(id);
        setVideo(data);
        setLocalLikes(data.likes);
        await incrementView(id);
      } catch (error) {
        console.error('Failed to fetch video:', error);
      } finally {
        setLoading(false);
      }
    };

    const fetchComments = async () => {
      try {
        const data = await getComments(id);
        setComments(data);
      } catch (error) {
        console.error('Failed to fetch comments:', error);
      }
    };

    const fetchPlaylist = async () => {
      const playlistId = searchParams.get('playlist');
      if (playlistId) {
        try {
          const playlistData = await getPlaylist(playlistId);
          setPlaylist(playlistData);
          
          // Fetch all videos to get details for playlist videos
          // Note: This could be optimized with a dedicated endpoint that returns
          // only the videos in a specific playlist, but for now we use the existing
          // videos endpoint and filter client-side for simplicity
          const allVideos = await getVideos();
          const playlistVids = playlistData.video_ids.map(vidId => 
            allVideos.find(v => v.id === vidId)
          ).filter(v => v !== undefined);
          setPlaylistVideos(playlistVids);
          
          // Find current video index in playlist
          const index = playlistData.video_ids.indexOf(parseInt(id));
          setCurrentVideoIndex(index);
        } catch (error) {
          console.error('Failed to fetch playlist:', error);
        }
      }
    };

    fetchVideo();
    fetchComments();
    fetchPlaylist();
  }, [id, searchParams]);

  useEffect(() => {
    if (videoRef.current) {
      videoRef.current.playbackRate = playbackSpeed;
    }
  }, [playbackSpeed]);

  // Handle video ended event for autoplay
  useEffect(() => {
    const videoElement = videoRef.current;
    if (!videoElement || !playlist) return;

    const handleVideoEnded = () => {
      // Check if there's a next video in the playlist
      if (currentVideoIndex >= 0 && currentVideoIndex < playlist.video_ids.length - 1) {
        const nextVideoId = playlist.video_ids[currentVideoIndex + 1];
        navigate(`/video/${nextVideoId}?playlist=${playlist.id}`);
      }
    };

    videoElement.addEventListener('ended', handleVideoEnded);
    return () => {
      videoElement.removeEventListener('ended', handleVideoEnded);
    };
  }, [playlist, currentVideoIndex, navigate]);

  const handleLike = async () => {
    try {
      const action = liked ? 'unlike' : 'like';
      await toggleLike(id, action);
      setLiked(!liked);
      setLocalLikes((prev) => (liked ? prev - 1 : prev + 1));
    } catch (error) {
      console.error('Failed to toggle like:', error);
    }
  };

  const handleFullscreen = () => {
    if (!isFullscreen) {
      const elem = containerRef.current;
      if (elem.requestFullscreen) {
        elem.requestFullscreen();
      } else if (elem.webkitRequestFullscreen) {
        elem.webkitRequestFullscreen();
      } else if (elem.mozRequestFullScreen) {
        elem.mozRequestFullScreen();
      } else if (elem.msRequestFullscreen) {
        elem.msRequestFullscreen();
      }
    } else {
      if (document.exitFullscreen) {
        document.exitFullscreen();
      } else if (document.webkitExitFullscreen) {
        document.webkitExitFullscreen();
      } else if (document.mozCancelFullScreen) {
        document.mozCancelFullScreen();
      } else if (document.msExitFullscreen) {
        document.msExitFullscreen();
      }
    }
  };

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(
        !!document.fullscreenElement ||
        !!document.webkitFullscreenElement ||
        !!document.mozFullScreenElement ||
        !!document.msFullscreenElement
      );
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('webkitfullscreenchange', handleFullscreenChange);
    document.addEventListener('mozfullscreenchange', handleFullscreenChange);
    document.addEventListener('MSFullscreenChange', handleFullscreenChange);

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      document.removeEventListener('webkitfullscreenchange', handleFullscreenChange);
      document.removeEventListener('mozfullscreenchange', handleFullscreenChange);
      document.removeEventListener('MSFullscreenChange', handleFullscreenChange);
    };
  }, []);

  useEffect(() => {
    const handleOrientationLock = async () => {
      if (isFullscreen && window.screen.orientation && window.screen.orientation.lock) {
        try {
          // Lock to current orientation when entering fullscreen
          await window.screen.orientation.lock(window.screen.orientation.type);
        } catch (error) {
          console.log('Orientation lock not supported or failed:', error);
        }
      } else if (!isFullscreen && window.screen.orientation && window.screen.orientation.unlock) {
        try {
          // Unlock orientation when exiting fullscreen
          window.screen.orientation.unlock();
        } catch (error) {
          console.log('Orientation unlock not supported or failed:', error);
        }
      }
    };

    handleOrientationLock();

    // Cleanup: unlock orientation when component unmounts
    return () => {
      if (window.screen.orientation && window.screen.orientation.unlock) {
        try {
          window.screen.orientation.unlock();
        } catch (error) {
          console.log('Orientation unlock cleanup failed:', error);
        }
      }
    };
  }, [isFullscreen]);

  const handleAddComment = async (author, content) => {
    try {
      const newComment = await addComment(id, author, content);
      setComments([newComment, ...comments]);
    } catch (error) {
      console.error('Failed to add comment:', error);
      throw error;
    }
  };

  const handlePlaylistVideoClick = (videoId) => {
    navigate(`/video/${videoId}?playlist=${playlist.id}`);
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="100vh">
        <CircularProgress />
      </Box>
    );
  }

  if (!video) {
    return (
      <Container>
        <Typography variant="h4" sx={{ mt: 4 }}>
          Video not found
        </Typography>
        <Button onClick={() => navigate('/')} sx={{ mt: 2 }}>
          Go Back
        </Button>
      </Container>
    );
  }

  return (
    <>
      <AppBar position="static">
        <Toolbar>
          <IconButton
            edge="start"
            color="inherit"
            aria-label="back"
            onClick={() => navigate('/')}
            sx={{ mr: 2 }}
          >
            <ArrowBack />
          </IconButton>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            StreamLite
          </Typography>
        </Toolbar>
      </AppBar>
      <Box sx={{ backgroundColor: '#000', minHeight: 'calc(100vh - 64px)' }}>
        {/* Theater Mode Container */}
        <Container maxWidth="xl" sx={{ pt: 2, pb: 4 }}>
          <Box sx={{ display: 'flex', gap: 2, flexDirection: { xs: 'column', lg: 'row' } }}>
            {/* Main Video Section */}
            <Box sx={{ flex: playlist ? '0 0 70%' : '1', minWidth: 0 }}>
              <Paper elevation={3} sx={{ mb: 2, backgroundColor: '#000' }} ref={containerRef}>
                <Box sx={{ position: 'relative', backgroundColor: '#000', width: '100%', height: isFullscreen ? '100vh' : 'auto' }}>
                  <video
                    ref={videoRef}
                    width="100%"
                    height={isFullscreen ? '100%' : 'auto'}
                    controls
                    src={getVideoStreamUrl(id)}
                    style={{ 
                      display: 'block',
                      objectFit: 'contain',
                      aspectRatio: '16/9',
                      maxHeight: isFullscreen ? '100vh' : '70vh'
                    }}
                  />
                  <Box
                    sx={{
                      position: 'absolute',
                      bottom: 60,
                      right: 10,
                      display: 'flex',
                      gap: 1,
                      backgroundColor: 'rgba(0,0,0,0.5)',
                      borderRadius: 1,
                      padding: 1,
                    }}
                  >
                    <FormControl size="small" sx={{ minWidth: 80 }}>
                      <InputLabel sx={{ color: 'white' }}>Speed</InputLabel>
                      <Select
                        value={playbackSpeed}
                        label="Speed"
                        onChange={(e) => setPlaybackSpeed(e.target.value)}
                        sx={{
                          color: 'white',
                          '.MuiOutlinedInput-notchedOutline': { borderColor: 'white' },
                          '&:hover .MuiOutlinedInput-notchedOutline': { borderColor: 'white' },
                          '.MuiSvgIcon-root': { color: 'white' },
                        }}
                      >
                        <MenuItem value={0.25}>0.25x</MenuItem>
                        <MenuItem value={0.5}>0.5x</MenuItem>
                        <MenuItem value={0.75}>0.75x</MenuItem>
                        <MenuItem value={1}>1x</MenuItem>
                        <MenuItem value={1.25}>1.25x</MenuItem>
                        <MenuItem value={1.5}>1.5x</MenuItem>
                        <MenuItem value={1.75}>1.75x</MenuItem>
                        <MenuItem value={2}>2x</MenuItem>
                        <MenuItem value={2.5}>2.5x</MenuItem>
                        <MenuItem value={3}>3x</MenuItem>
                      </Select>
                    </FormControl>
                    <IconButton
                      onClick={handleFullscreen}
                      sx={{ color: 'white' }}
                      aria-label="fullscreen"
                    >
                      {isFullscreen ? <FullscreenExit /> : <Fullscreen />}
                    </IconButton>
                  </Box>
                </Box>
              </Paper>
              
              {/* Video Info */}
              <Paper elevation={3} sx={{ p: 2, mb: 2, backgroundColor: '#1e1e1e', color: 'white' }}>
                <Typography variant="h5" gutterBottom>
                  {video?.title}
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
                  <Typography variant="body2" color="grey.400">
                    {video?.views} views
                  </Typography>
                  <Button
                    variant={liked ? 'contained' : 'outlined'}
                    startIcon={liked ? <ThumbUp /> : <ThumbUpOutlined />}
                    onClick={handleLike}
                  >
                    {localLikes}
                  </Button>
                </Box>
              </Paper>

              {/* Comments Section */}
              <Box sx={{ backgroundColor: '#fff' }}>
                <CommentSection comments={comments} onAddComment={handleAddComment} />
              </Box>
            </Box>

            {/* Playlist Sidebar */}
            {playlist && playlistVideos.length > 0 && (
              <Box sx={{ flex: '0 0 28%', minWidth: 0 }}>
                <Paper elevation={3} sx={{ p: 2, backgroundColor: '#1e1e1e', color: 'white', position: 'sticky', top: 16, maxHeight: 'calc(100vh - 100px)', overflow: 'auto' }}>
                  <Typography variant="h6" gutterBottom>
                    {playlist.name}
                  </Typography>
                  <Typography variant="body2" color="grey.400" sx={{ mb: 2 }}>
                    {currentVideoIndex + 1} / {playlist.video_count} videos
                  </Typography>
                  <Divider sx={{ mb: 2, backgroundColor: 'grey.700' }} />
                  <List sx={{ p: 0 }}>
                    {playlistVideos.map((playlistVideo, index) => (
                      <ListItem key={playlistVideo.id} disablePadding sx={{ mb: 1 }}>
                        <ListItemButton
                          onClick={() => handlePlaylistVideoClick(playlistVideo.id)}
                          selected={playlistVideo.id === parseInt(id)}
                          sx={{
                            borderRadius: 1,
                            backgroundColor: playlistVideo.id === parseInt(id) ? 'primary.main' : 'transparent',
                            '&:hover': {
                              backgroundColor: playlistVideo.id === parseInt(id) ? 'primary.dark' : 'rgba(255, 255, 255, 0.08)',
                            },
                            p: 1,
                          }}
                        >
                          <Box sx={{ display: 'flex', gap: 1, width: '100%' }}>
                            <Typography variant="body2" sx={{ minWidth: 24, color: 'grey.400' }}>
                              {index + 1}
                            </Typography>
                            <Box sx={{ flex: 1, minWidth: 0 }}>
                              <CardMedia
                                component="img"
                                image={playlistVideo.thumbnail_url}
                                alt={playlistVideo.title}
                                sx={{
                                  width: '100%',
                                  aspectRatio: '16/9',
                                  objectFit: 'cover',
                                  borderRadius: 1,
                                  mb: 0.5,
                                }}
                              />
                              <Typography 
                                variant="body2" 
                                sx={{ 
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis',
                                  display: '-webkit-box',
                                  WebkitLineClamp: 2,
                                  WebkitBoxOrient: 'vertical',
                                }}
                              >
                                {playlistVideo.title}
                              </Typography>
                            </Box>
                          </Box>
                        </ListItemButton>
                      </ListItem>
                    ))}
                  </List>
                </Paper>
              </Box>
            )}
          </Box>
        </Container>
      </Box>
    </>
  );
};

export default VideoPlayerPage;
