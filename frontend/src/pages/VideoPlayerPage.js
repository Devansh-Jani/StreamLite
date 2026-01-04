import React, { useEffect, useState, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
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
} from '../api';
import CommentSection from '../components/CommentSection';

const VideoPlayerPage = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const videoRef = useRef(null);
  const containerRef = useRef(null);
  const [video, setVideo] = useState(null);
  const [loading, setLoading] = useState(true);
  const [liked, setLiked] = useState(false);
  const [localLikes, setLocalLikes] = useState(0);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const [comments, setComments] = useState([]);

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

    fetchVideo();
    fetchComments();
  }, [id]);

  useEffect(() => {
    if (videoRef.current) {
      videoRef.current.playbackRate = playbackSpeed;
    }
  }, [playbackSpeed]);

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
      if (isFullscreen && screen.orientation && screen.orientation.lock) {
        try {
          // Lock to current orientation when entering fullscreen
          await screen.orientation.lock(screen.orientation.type);
        } catch (error) {
          console.log('Orientation lock not supported or failed:', error);
        }
      } else if (!isFullscreen && screen.orientation && screen.orientation.unlock) {
        try {
          // Unlock orientation when exiting fullscreen
          screen.orientation.unlock();
        } catch (error) {
          console.log('Orientation unlock not supported or failed:', error);
        }
      }
    };

    handleOrientationLock();
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
      <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
        <Paper elevation={3} sx={{ mb: 3 }} ref={containerRef}>
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
                maxHeight: isFullscreen ? '100vh' : '80vh'
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
          <Box sx={{ p: 2 }}>
            <Typography variant="h5" gutterBottom>
              {video.title}
            </Typography>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
              <Typography variant="body2" color="text.secondary">
                {video.views} views
              </Typography>
              <Button
                variant={liked ? 'contained' : 'outlined'}
                startIcon={liked ? <ThumbUp /> : <ThumbUpOutlined />}
                onClick={handleLike}
              >
                {localLikes}
              </Button>
            </Box>
          </Box>
        </Paper>
        <CommentSection comments={comments} onAddComment={handleAddComment} />
      </Container>
    </>
  );
};

export default VideoPlayerPage;
