import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Box,
  Alert,
} from '@mui/material';
import { Vehicle } from '../types';

interface VehicleFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (vehicle: Omit<Vehicle, 'id'>) => Promise<void>;
  vehicle?: Vehicle;
  title: string;
}

const VehicleForm: React.FC<VehicleFormProps> = ({
  open,
  onClose,
  onSubmit,
  vehicle,
  title,
}) => {
  const [formData, setFormData] = useState<Omit<Vehicle, 'id'>>({
    type: 'ICE',
    make: '',
    model: '',
    year: new Date().getFullYear(),
    current_location: { lat: 40.7128, lon: -74.0060 },
    status: 'active',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (vehicle) {
      setFormData({
        type: vehicle.type,
        make: vehicle.make || '',
        model: vehicle.model || '',
        year: vehicle.year || new Date().getFullYear(),
        current_location: vehicle.current_location || { lat: 40.7128, lon: -74.0060 },
        status: vehicle.status,
      });
    } else {
      setFormData({
        type: 'ICE',
        make: '',
        model: '',
        year: new Date().getFullYear(),
        current_location: { lat: 40.7128, lon: -74.0060 },
        status: 'active',
      });
    }
    setErrors({});
  }, [vehicle, open]);

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.make?.trim()) {
      newErrors.make = 'Make is required';
    }
    if (!formData.model?.trim()) {
      newErrors.model = 'Model is required';
    }
    if (!formData.year || formData.year < 1900 || formData.year > 2030) {
      newErrors.year = 'Year must be between 1900 and 2030';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }

    setLoading(true);
    try {
      await onSubmit(formData);
      onClose();
    } catch (error) {
      console.error('Error submitting vehicle:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (field: keyof Omit<Vehicle, 'id'>, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <form onSubmit={handleSubmit}>
        <DialogContent>
          <Box display="flex" flexDirection="column" gap={2}>
            <FormControl fullWidth>
              <InputLabel>Vehicle Type</InputLabel>
              <Select
                value={formData.type}
                onChange={(e) => handleChange('type', e.target.value)}
                label="Vehicle Type"
              >
                <MenuItem value="ICE">ICE (Internal Combustion Engine)</MenuItem>
                <MenuItem value="EV">EV (Electric Vehicle)</MenuItem>
              </Select>
            </FormControl>

            <TextField
              label="Make"
              value={formData.make}
              onChange={(e) => handleChange('make', e.target.value)}
              error={!!errors.make}
              helperText={errors.make}
              fullWidth
              required
            />

            <TextField
              label="Model"
              value={formData.model}
              onChange={(e) => handleChange('model', e.target.value)}
              error={!!errors.model}
              helperText={errors.model}
              fullWidth
              required
            />

            <TextField
              label="Year"
              type="number"
              value={formData.year}
              onChange={(e) => handleChange('year', parseInt(e.target.value))}
              error={!!errors.year}
              helperText={errors.year}
              fullWidth
              required
              inputProps={{ min: 1900, max: 2030 }}
            />

            <FormControl fullWidth>
              <InputLabel>Status</InputLabel>
              <Select
                value={formData.status}
                onChange={(e) => handleChange('status', e.target.value)}
                label="Status"
              >
                <MenuItem value="active">Active</MenuItem>
                <MenuItem value="inactive">Inactive</MenuItem>
              </Select>
            </FormControl>

            <Box>
              <TextField
                label="Latitude"
                type="number"
                value={formData.current_location?.lat || ''}
                onChange={(e) => handleChange('current_location', {
                  ...formData.current_location,
                  lat: parseFloat(e.target.value)
                })}
                fullWidth
                inputProps={{ step: 0.000001 }}
              />
            </Box>

            <Box>
              <TextField
                label="Longitude"
                type="number"
                value={formData.current_location?.lon || ''}
                onChange={(e) => handleChange('current_location', {
                  ...formData.current_location,
                  lon: parseFloat(e.target.value)
                })}
                fullWidth
                inputProps={{ step: 0.000001 }}
              />
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" disabled={loading}>
            {loading ? 'Saving...' : 'Save'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default VehicleForm; 