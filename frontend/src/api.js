import axios from 'axios';

// Dynamically determine API base URL
// If REACT_APP_API_URL is set, use it
// Otherwise, use the current host's protocol and hostname with /api path
const getApiBaseUrl = () => {
  if (process.env.REACT_APP_API_URL) {
    return process.env.REACT_APP_API_URL;
  }
  
  // In production (when served via nginx), use relative path
  // In development, fall back to localhost
  if (process.env.NODE_ENV === 'production') {
    return `${window.location.protocol}//${window.location.host}/api`;
  }
  
  return 'http://localhost:8082/api';
};

const API_BASE_URL = getApiBaseUrl();

export const getVideos = async () => {
  const response = await axios.get(`${API_BASE_URL}/videos`);
  return response.data;
};

export const refreshVideos = async () => {
  const response = await axios.post(`${API_BASE_URL}/videos/refresh`);
  return response.data;
};

export const getVideo = async (id) => {
  const response = await axios.get(`${API_BASE_URL}/videos/${id}`);
  return response.data;
};

export const getVideoStreamUrl = (id) => {
  return `${API_BASE_URL}/videos/${id}/stream`;
};

export const incrementView = async (id) => {
  await axios.post(`${API_BASE_URL}/videos/${id}/view`);
};

export const toggleLike = async (id, action) => {
  await axios.post(`${API_BASE_URL}/videos/${id}/like`, { action });
};

export const getComments = async (id) => {
  const response = await axios.get(`${API_BASE_URL}/videos/${id}/comments`);
  return response.data;
};

export const addComment = async (id, author, content) => {
  const response = await axios.post(`${API_BASE_URL}/videos/${id}/comments`, {
    author,
    content,
  });
  return response.data;
};
