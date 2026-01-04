import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8082/api';

export const getVideos = async () => {
  const response = await axios.get(`${API_BASE_URL}/videos`);
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
