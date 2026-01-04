import React, { useState } from 'react';
import {
  Paper,
  Typography,
  TextField,
  Button,
  Box,
  Avatar,
  List,
  ListItem,
  ListItemAvatar,
  ListItemText,
} from '@mui/material';

const CommentSection = ({ comments, onAddComment }) => {
  const [author, setAuthor] = useState('');
  const [content, setContent] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!content.trim()) {
      return;
    }

    setSubmitting(true);
    try {
      await onAddComment(author || 'Anonymous', content);
      setContent('');
      setAuthor('');
    } catch (error) {
      console.error('Failed to submit comment:', error);
    } finally {
      setSubmitting(false);
    }
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffTime = Math.abs(now - date);
    const diffMinutes = Math.floor(diffTime / (1000 * 60));
    const diffHours = Math.floor(diffTime / (1000 * 60 * 60));
    const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));

    if (diffMinutes < 1) {
      return 'Just now';
    } else if (diffMinutes < 60) {
      return `${diffMinutes} minute${diffMinutes > 1 ? 's' : ''} ago`;
    } else if (diffHours < 24) {
      return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`;
    } else if (diffDays < 7) {
      return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`;
    } else {
      return date.toLocaleDateString();
    }
  };

  return (
    <Paper elevation={3} sx={{ p: 3 }}>
      <Typography variant="h6" gutterBottom>
        Comments ({comments.length})
      </Typography>
      <Box component="form" onSubmit={handleSubmit} sx={{ mb: 3 }}>
        <TextField
          fullWidth
          label="Your Name (optional)"
          value={author}
          onChange={(e) => setAuthor(e.target.value)}
          margin="normal"
          size="small"
        />
        <TextField
          fullWidth
          label="Add a comment..."
          value={content}
          onChange={(e) => setContent(e.target.value)}
          margin="normal"
          multiline
          rows={3}
          required
        />
        <Button
          type="submit"
          variant="contained"
          sx={{ mt: 1 }}
          disabled={submitting || !content.trim()}
        >
          {submitting ? 'Posting...' : 'Post Comment'}
        </Button>
      </Box>
      <List>
        {comments.map((comment) => (
          <ListItem key={comment.id} alignItems="flex-start" sx={{ px: 0 }}>
            <ListItemAvatar>
              <Avatar>{comment.author.charAt(0).toUpperCase()}</Avatar>
            </ListItemAvatar>
            <ListItemText
              primary={
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Typography variant="subtitle2">{comment.author}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {formatDate(comment.created_at)}
                  </Typography>
                </Box>
              }
              secondary={<Typography variant="body2">{comment.content}</Typography>}
            />
          </ListItem>
        ))}
      </List>
    </Paper>
  );
};

export default CommentSection;
