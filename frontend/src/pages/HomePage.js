import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Container,
  Grid,
  Card,
  CardMedia,
  CardContent,
  Typography,
  AppBar,
  Toolbar,
  Box,
  CircularProgress,
  IconButton,
  Tooltip,
  Snackbar,
  Alert,
  Chip,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import PlaylistPlayIcon from '@mui/icons-material/PlaylistPlay';
import { getVideos, refreshVideos, getPlaylists } from '../api';

const HomePage = () => {
  const [videos, setVideos] = useState([]);
  const [playlists, setPlaylists] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });
  const navigate = useNavigate();

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [videosData, playlistsData] = await Promise.all([
          getVideos(),
          getPlaylists(),
        ]);
        setVideos(videosData);
        setPlaylists(playlistsData);
      } catch (error) {
        console.error('Failed to fetch data:', error);
        setSnackbar({ open: true, message: 'Failed to load videos', severity: 'error' });
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await refreshVideos();
      const [videosData, playlistsData] = await Promise.all([
        getVideos(),
        getPlaylists(),
      ]);
      setVideos(videosData);
      setPlaylists(playlistsData);
      setSnackbar({ open: true, message: 'Videos refreshed successfully', severity: 'success' });
    } catch (error) {
      console.error('Failed to refresh videos:', error);
      setSnackbar({ open: true, message: 'Failed to refresh videos', severity: 'error' });
    } finally {
      setRefreshing(false);
    }
  };

  const handleSnackbarClose = () => {
    setSnackbar({ ...snackbar, open: false });
  };

  const handleVideoClick = (id) => {
    navigate(`/video/${id}`);
  };

  const handlePlaylistClick = (playlist) => {
    // Navigate to the first video with playlist info
    navigate(`/video/${playlist.video_ids[0]}?playlist=${playlist.id}`);
  };

  const formatViews = (views) => {
    if (views >= 1000000) {
      return `${(views / 1000000).toFixed(1)}M`;
    } else if (views >= 1000) {
      return `${(views / 1000).toFixed(1)}K`;
    }
    return views.toString();
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffTime = Math.abs(now - date);
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays === 0) {
      return 'Today';
    } else if (diffDays === 1) {
      return 'Yesterday';
    } else if (diffDays < 7) {
      return `${diffDays} days ago`;
    } else if (diffDays < 30) {
      return `${Math.floor(diffDays / 7)} weeks ago`;
    } else if (diffDays < 365) {
      return `${Math.floor(diffDays / 30)} months ago`;
    } else {
      return `${Math.floor(diffDays / 365)} years ago`;
    }
  };

  return (
    <>
      <AppBar position="static">
        <Toolbar>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            StreamLite
          </Typography>
          <Tooltip title="Refresh videos">
            <IconButton 
              color="inherit" 
              onClick={handleRefresh}
              disabled={refreshing}
            >
              {refreshing ? <CircularProgress size={24} color="inherit" /> : <RefreshIcon />}
            </IconButton>
          </Tooltip>
        </Toolbar>
      </AppBar>
      <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
        {loading ? (
          <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
            <CircularProgress />
          </Box>
        ) : (
          <>
            {playlists.length > 0 && (
              <Box sx={{ mb: 4 }}>
                <Typography variant="h5" gutterBottom sx={{ mb: 2, fontWeight: 'bold' }}>
                  Playlists
                </Typography>
                <Grid container spacing={3}>
                  {playlists.map((playlist) => (
                    <Grid item xs={12} sm={6} md={4} key={playlist.id}>
                      <Card
                        sx={{ 
                          cursor: 'pointer', 
                          height: '100%', 
                          display: 'flex', 
                          flexDirection: 'column',
                          border: '2px solid',
                          borderColor: 'primary.main',
                        }}
                        onClick={() => handlePlaylistClick(playlist)}
                      >
                        <Box sx={{ position: 'relative' }}>
                          <CardMedia
                            component="img"
                            image={`/api/videos/${playlist.thumbnail_id}/thumbnail`}
                            alt={playlist.name}
                            sx={{
                              aspectRatio: '16/9',
                              objectFit: 'cover',
                              backgroundColor: '#000',
                            }}
                          />
                          <Box
                            sx={{
                              position: 'absolute',
                              top: 8,
                              right: 8,
                              backgroundColor: 'rgba(0, 0, 0, 0.8)',
                              borderRadius: 1,
                              px: 1,
                              py: 0.5,
                              display: 'flex',
                              alignItems: 'center',
                              gap: 0.5,
                            }}
                          >
                            <PlaylistPlayIcon sx={{ color: 'white', fontSize: 20 }} />
                            <Typography variant="body2" sx={{ color: 'white', fontWeight: 'bold' }}>
                              {playlist.video_count}
                            </Typography>
                          </Box>
                        </Box>
                        <CardContent sx={{ flexGrow: 1 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                            <PlaylistPlayIcon color="primary" />
                            <Chip label="Playlist" color="primary" size="small" />
                          </Box>
                          <Typography gutterBottom variant="h6" component="div">
                            {playlist.name}
                          </Typography>
                          <Typography variant="body2" color="text.secondary">
                            {playlist.video_count} videos
                          </Typography>
                        </CardContent>
                      </Card>
                    </Grid>
                  ))}
                </Grid>
              </Box>
            )}

            <Typography variant="h5" gutterBottom sx={{ mb: 2, fontWeight: 'bold' }}>
              All Videos
            </Typography>
            <Grid container spacing={3}>
              {videos.map((video) => (
                <Grid item xs={12} sm={6} md={4} key={video.id}>
                  <Card
                    sx={{ cursor: 'pointer', height: '100%', display: 'flex', flexDirection: 'column' }}
                    onClick={() => handleVideoClick(video.id)}
                  >
                    <CardMedia
                      component="img"
                      image={video.thumbnail_url}
                      alt={video.title}
                      sx={{
                        aspectRatio: '16/9',
                        objectFit: 'cover',
                        backgroundColor: '#000',
                      }}
                    />
                    <CardContent sx={{ flexGrow: 1 }}>
                      <Typography gutterBottom variant="h6" component="div" noWrap>
                        {video.title}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        {formatViews(video.views)} views • {formatDate(video.modified_at)}
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          </>
        )}
      </Container>
      <Snackbar 
        open={snackbar.open} 
        autoHideDuration={4000} 
        onClose={handleSnackbarClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={handleSnackbarClose} severity={snackbar.severity} sx={{ width: '100%' }}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </>
  );
};

export default HomePage;
