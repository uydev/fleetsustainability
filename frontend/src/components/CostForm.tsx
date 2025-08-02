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
} from '@mui/material';
import { DatePicker } from '@mui/x-date-pickers/DatePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { Cost } from '../types';

interface CostFormProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (cost: Omit<Cost, 'id'>) => Promise<void>;
  title: string;
}

const CostForm: React.FC<CostFormProps> = ({
  open,
  onClose,
  onSubmit,
  title,
}) => {
  const [formData, setFormData] = useState<Omit<Cost, 'id'>>({
    vehicle_id: '',
    category: 'fuel',
    description: '',
    amount: 0,
    date: new Date().toISOString(),
    invoice_number: '',
    vendor: '',
    location: '',
    payment_method: 'credit_card',
    status: 'pending',
    receipt_url: '',
    notes: '',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setFormData({
        vehicle_id: '',
        category: 'fuel',
        description: '',
        amount: 0,
        date: new Date().toISOString(),
        invoice_number: '',
        vendor: '',
        location: '',
        payment_method: 'credit_card',
        status: 'pending',
        receipt_url: '',
        notes: '',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      });
      setErrors({});
    }
  }, [open]);

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.vehicle_id.trim()) {
      newErrors.vehicle_id = 'Vehicle ID is required';
    }
    if (!formData.description.trim()) {
      newErrors.description = 'Description is required';
    }
    if (formData.amount <= 0) {
      newErrors.amount = 'Amount must be positive';
    }
    if (!formData.vendor.trim()) {
      newErrors.vendor = 'Vendor is required';
    }
    if (!formData.location.trim()) {
      newErrors.location = 'Location is required';
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
      console.error('Error submitting cost:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleChange = (field: keyof Omit<Cost, 'id'>, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <form onSubmit={handleSubmit}>
        <DialogContent>
          <Box display="flex" flexDirection="column" gap={2}>
            <TextField
              label="Vehicle ID"
              value={formData.vehicle_id}
              onChange={(e) => handleChange('vehicle_id', e.target.value)}
              error={!!errors.vehicle_id}
              helperText={errors.vehicle_id}
              fullWidth
              required
            />

            <FormControl fullWidth>
              <InputLabel>Category</InputLabel>
              <Select
                value={formData.category}
                onChange={(e) => handleChange('category', e.target.value)}
                label="Category"
              >
                <MenuItem value="fuel">Fuel</MenuItem>
                <MenuItem value="maintenance">Maintenance</MenuItem>
                <MenuItem value="insurance">Insurance</MenuItem>
                <MenuItem value="registration">Registration</MenuItem>
                <MenuItem value="tolls">Tolls</MenuItem>
                <MenuItem value="parking">Parking</MenuItem>
                <MenuItem value="other">Other</MenuItem>
              </Select>
            </FormControl>

            <TextField
              label="Description"
              value={formData.description}
              onChange={(e) => handleChange('description', e.target.value)}
              error={!!errors.description}
              helperText={errors.description}
              fullWidth
              multiline
              rows={2}
              required
            />

            <TextField
              label="Amount ($)"
              type="number"
              value={formData.amount}
              onChange={(e) => handleChange('amount', parseFloat(e.target.value))}
              error={!!errors.amount}
              helperText={errors.amount}
              fullWidth
              required
              inputProps={{ min: 0, step: 0.01 }}
            />

            <LocalizationProvider dateAdapter={AdapterDateFns}>
              <DatePicker
                label="Date"
                value={new Date(formData.date)}
                onChange={(newValue) => {
                  if (newValue) {
                    handleChange('date', newValue.toISOString());
                  }
                }}
                slotProps={{
                  textField: {
                    fullWidth: true,
                    required: true,
                  },
                }}
              />
            </LocalizationProvider>

            <TextField
              label="Invoice Number"
              value={formData.invoice_number}
              onChange={(e) => handleChange('invoice_number', e.target.value)}
              fullWidth
            />

            <TextField
              label="Vendor"
              value={formData.vendor}
              onChange={(e) => handleChange('vendor', e.target.value)}
              error={!!errors.vendor}
              helperText={errors.vendor}
              fullWidth
              required
            />

            <TextField
              label="Location"
              value={formData.location}
              onChange={(e) => handleChange('location', e.target.value)}
              error={!!errors.location}
              helperText={errors.location}
              fullWidth
              required
            />

            <Box display="flex" gap={2}>
              <FormControl fullWidth>
                <InputLabel>Payment Method</InputLabel>
                <Select
                  value={formData.payment_method}
                  onChange={(e) => handleChange('payment_method', e.target.value)}
                  label="Payment Method"
                >
                  <MenuItem value="credit_card">Credit Card</MenuItem>
                  <MenuItem value="cash">Cash</MenuItem>
                  <MenuItem value="check">Check</MenuItem>
                  <MenuItem value="electronic">Electronic</MenuItem>
                </Select>
              </FormControl>

              <FormControl fullWidth>
                <InputLabel>Status</InputLabel>
                <Select
                  value={formData.status}
                  onChange={(e) => handleChange('status', e.target.value)}
                  label="Status"
                >
                  <MenuItem value="pending">Pending</MenuItem>
                  <MenuItem value="paid">Paid</MenuItem>
                  <MenuItem value="disputed">Disputed</MenuItem>
                  <MenuItem value="cancelled">Cancelled</MenuItem>
                </Select>
              </FormControl>
            </Box>

            <TextField
              label="Receipt URL"
              value={formData.receipt_url}
              onChange={(e) => handleChange('receipt_url', e.target.value)}
              fullWidth
              placeholder="https://example.com/receipt"
            />

            <TextField
              label="Notes"
              value={formData.notes}
              onChange={(e) => handleChange('notes', e.target.value)}
              fullWidth
              multiline
              rows={3}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button type="submit" variant="contained" disabled={loading}>
            {loading ? 'Saving...' : 'Save Cost'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default CostForm; 