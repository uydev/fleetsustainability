import React, { useEffect, useMemo, useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Button,
  Grid,
  Card,
  CardContent,
  Alert,
  LinearProgress,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemText,
  Divider,
  Snackbar,
} from '@mui/material';
import {
  TrendingUp,
  ElectricCar,
  LocalGasStation,
  AttachMoney,
  Park,
  Info,
} from '@mui/icons-material';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip as RechartsTooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import jsPDF from 'jspdf';
import apiService from '../services/api';
import { Vehicle, Telemetry } from '../types';

interface ElectrificationRecommendation {
  vehicleId: string;
  vehicleType: string;
  priority: 'High' | 'Medium' | 'Low';
  score: number;
  annualEmissions: number;
  annualFuelCost: number;
  projectedEvCost: number;
  annualSavings: number;
  co2Reduction: number;
  paybackPeriod: number; // in years
  reasons: string[];
}

interface ElectrificationSummary {
  totalIceVehicles: number;
  recommendedReplacements: number;
  totalAnnualSavings: number;
  totalCo2Reduction: number;
  averagePaybackPeriod: number;
}

type Props = {
  vehicles: Vehicle[];
  timeRange?: { from?: string; to?: string };
};

const ElectrificationPlanning: React.FC<Props> = ({ vehicles, timeRange }) => {
  const [telemetry, setTelemetry] = useState<Telemetry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [planDialogOpen, setPlanDialogOpen] = useState(false);
  const [investmentDialogOpen, setInvestmentDialogOpen] = useState(false);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [snackbarMessage, setSnackbarMessage] = useState('');

  // Assumptions (could be moved to settings later)
  const FUEL_PRICE_PER_LITER = 1.5; // €
  const EV_PRICE = 35000; // €
  const ELECTRICITY_PRICE_PER_KWH = 0.25; // €
  const TANK_CAPACITY_L = 50; // liters
  const BATTERY_CAPACITY_KWH = 60; // kWh
  const EV_EFFICIENCY_KWH_PER_KM = 0.18; // kWh per km (typical compact EV)
  const MAINT_ICE_PER_KM = 0.08; // € per km
  const MAINT_EV_PER_KM = 0.05; // € per km
  const EV_INCENTIVE = 5000; // € government incentive

  const toMs = (s: string) => new Date(s).getTime();
  const kmBetween = (a: { lat: number; lon: number }, b: { lat: number; lon: number }) => {
    const R = 6371; // km
    const dLat = (b.lat - a.lat) * Math.PI / 180;
    const dLon = (b.lon - a.lon) * Math.PI / 180;
    const lat1 = a.lat * Math.PI / 180;
    const lat2 = b.lat * Math.PI / 180;
    const sinDLat = Math.sin(dLat / 2);
    const sinDLon = Math.sin(dLon / 2);
    const h = sinDLat * sinDLat + Math.cos(lat1) * Math.cos(lat2) * sinDLon * sinDLon;
    return 2 * R * Math.asin(Math.sqrt(h));
  };

  const computeDistanceKm = (rows: Telemetry[]): number => {
    let dist = 0;
    for (let i = 1; i < rows.length; i++) {
      const prev = rows[i - 1];
      const curr = rows[i];
      dist += kmBetween(prev.location, curr.location);
    }
    return dist;
  };

  const computeDaysCovered = (rows: Telemetry[]): number => {
    if (rows.length < 2) return 1;
    const days = (toMs(rows[rows.length - 1].timestamp) - toMs(rows[0].timestamp)) / (1000 * 60 * 60 * 24);
    return Math.max(1, days);
  };

  const computeFuelUsedLiters = (rows: Telemetry[]): number => {
    // Sum only decreases in fuel_level to approximate consumption
    let usedPct = 0;
    for (let i = 1; i < rows.length; i++) {
      const prev = rows[i - 1];
      const curr = rows[i];
      const p = prev.fuel_level ?? null;
      const c = curr.fuel_level ?? null;
      if (p !== null && c !== null) {
        const drop = p - c;
        if (drop > 0) usedPct += drop;
      }
    }
    return (usedPct / 100) * TANK_CAPACITY_L;
  };

  // computeEnergyUsedKwhFromBattery was unused – remove to satisfy linter

  // Fetch telemetry data
  useEffect(() => {
    const fetchTelemetry = async () => {
      try {
        setLoading(true);
        const data = await apiService.getTelemetry(timeRange);
        setTelemetry(data);
      } catch (err) {
        setError('Failed to fetch telemetry data');
        console.error('Error fetching telemetry:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchTelemetry();
  }, [timeRange]);

  // Calculate electrification recommendations
  const recommendations = useMemo((): ElectrificationRecommendation[] => {
    const iceVehicles = vehicles.filter(v => v.type === 'ICE');
    
    return iceVehicles.map(vehicle => {
      const vehicleTelemetry = telemetry
        .filter(t => t.vehicle_id === vehicle.id)
        .sort((a, b) => toMs(a.timestamp) - toMs(b.timestamp));
      
      if (vehicleTelemetry.length === 0) {
        return {
          vehicleId: vehicle.id,
          vehicleType: vehicle.type,
          priority: 'Low' as const,
          score: 0,
          annualEmissions: 0,
          annualFuelCost: 0,
          projectedEvCost: 0,
          annualSavings: 0,
          co2Reduction: 0,
          paybackPeriod: 0,
          reasons: ['No telemetry data available']
        };
      }

      // Calculate metrics from telemetry
      const daysOfData = computeDaysCovered(vehicleTelemetry);
      const distanceKm = computeDistanceKm(vehicleTelemetry);
      const avgSpeed = vehicleTelemetry.reduce((sum, t) => sum + t.speed, 0) / vehicleTelemetry.length;
      const totalEmissions = vehicleTelemetry.reduce((sum, t) => sum + t.emissions, 0);
      const emissionsPerKm = distanceKm > 0 ? totalEmissions / distanceKm : totalEmissions;

      // Annualize key metrics
      const annualKm = (distanceKm / daysOfData) * 365;
      const fuelUsedLiters = computeFuelUsedLiters(vehicleTelemetry);
      const annualFuelLiters = (fuelUsedLiters / daysOfData) * 365;
      const annualFuelCost = annualFuelLiters * FUEL_PRICE_PER_LITER;

      // EV operating cost (two approaches; use distance-based to avoid battery% assumptions on ICE)
      const annualEvKwh = annualKm * EV_EFFICIENCY_KWH_PER_KM;
      const annualEvCost = annualEvKwh * ELECTRICITY_PRICE_PER_KWH;

      // Maintenance costs
      const maintIce = annualKm * MAINT_ICE_PER_KM;
      const maintEv = annualKm * MAINT_EV_PER_KM;

      const annualSavings = (annualFuelCost + maintIce) - (annualEvCost + maintEv);
      const annualEmissions = (totalEmissions / daysOfData) * 365;
      const co2Reduction = Math.max(0, annualEmissions * 0.8);
      
      // Calculate priority score (0-100)
      let score = 0;
      const reasons: string[] = [];
      
      // Base score for ICE vehicles (all ICE vehicles should have some priority)
      if (vehicle.type === 'ICE') {
        score += 15;
        reasons.push('ICE vehicle eligible for replacement');
      }
      
      // High emissions per km = higher priority
      if (emissionsPerKm > 200) {
        score += 25;
        reasons.push('High emissions (>200g/km)');
      } else if (emissionsPerKm > 150) {
        score += 15;
        reasons.push('Moderate emissions (150-200g/km)');
      } else if (emissionsPerKm > 100) {
        score += 10;
        reasons.push('Some emissions (100-150g/km)');
      }
      
      // High annual kilometers -> higher priority
      if (annualKm > 30000) { score += 20; reasons.push('Very high annual mileage (>30k km)'); }
      else if (annualKm > 20000) { score += 15; reasons.push('High annual mileage (20-30k km)'); }
      else if (annualKm > 12000) { score += 10; reasons.push('Moderate annual mileage (12-20k km)'); }
      
      // Savings potential
      if (annualSavings > 3500) { score += 20; reasons.push('Very high savings potential (>€3500/year)'); }
      else if (annualSavings > 2000) { score += 15; reasons.push('High savings potential (€2000-3500/year)'); }
      else if (annualSavings > 800) { score += 10; reasons.push('Moderate savings potential (€800-2000/year)'); }
      
      // Age factor (older vehicles = higher priority)
      const vehicleAge = new Date().getFullYear() - (vehicle.year || 2020);
      if (vehicleAge > 10) {
        score += 15;
        reasons.push('Older vehicle (>10 years)');
      } else if (vehicleAge > 5) {
        score += 10;
        reasons.push('Mature vehicle (5-10 years)');
      }
      
      // Range feasibility (max daily km vs assumed EV range)
      let maxDailyKm = 0;
      if (vehicleTelemetry.length > 1) {
        let currentDay = new Date(vehicleTelemetry[0].timestamp).toDateString();
        let acc = 0;
        for (let i = 1; i < vehicleTelemetry.length; i++) {
          const prev = vehicleTelemetry[i - 1];
          const curr = vehicleTelemetry[i];
          const day = new Date(curr.timestamp).toDateString();
          const seg = kmBetween(prev.location, curr.location);
          if (day !== currentDay) {
            maxDailyKm = Math.max(maxDailyKm, acc);
            currentDay = day;
            acc = 0;
          }
          acc += seg;
        }
        maxDailyKm = Math.max(maxDailyKm, acc);
      }
      const ASSUMED_EV_RANGE_KM = 400;
      if (maxDailyKm > ASSUMED_EV_RANGE_KM) {
        reasons.push(`Occasional long days (${Math.round(maxDailyKm)} km) may require charging strategy`);
        score -= 5; // small penalty
      }

      const effectiveEvPrice = Math.max(0, EV_PRICE - EV_INCENTIVE);
      const paybackPeriod = annualSavings > 0 ? effectiveEvPrice / annualSavings : 0;
      
      // Payback period factor (shorter payback = higher priority)
      if (paybackPeriod > 0 && paybackPeriod < 5) {
        score += 15;
        reasons.push('Quick payback period (<5 years)');
      } else if (paybackPeriod > 0 && paybackPeriod < 10) {
        score += 10;
        reasons.push('Reasonable payback period (5-10 years)');
      }
      
      let priority: 'High' | 'Medium' | 'Low';
      if (score >= 45) priority = 'High';
      else if (score >= 25) priority = 'Medium';
      else priority = 'Low';
      
      
      return {
        vehicleId: vehicle.id,
        vehicleType: vehicle.type,
        priority,
        score,
        annualEmissions,
        annualFuelCost,
        projectedEvCost: EV_PRICE,
        annualSavings,
        co2Reduction,
        paybackPeriod,
        reasons
      };
    }).sort((a, b) => b.score - a.score);
  }, [vehicles, telemetry, computeDaysCovered, computeDistanceKm]);

  // Calculate summary statistics
  const summary = useMemo((): ElectrificationSummary => {
    const iceVehicles = vehicles.filter(v => v.type === 'ICE');
    const recommended = recommendations.filter(r => r.priority === 'High' || r.priority === 'Medium');
    
    return {
      totalIceVehicles: iceVehicles.length,
      recommendedReplacements: recommended.length,
      totalAnnualSavings: recommended.reduce((sum, r) => sum + r.annualSavings, 0),
      totalCo2Reduction: recommended.reduce((sum, r) => sum + r.co2Reduction, 0),
      averagePaybackPeriod: recommended.length > 0 ? 
        recommended.reduce((sum, r) => sum + r.paybackPeriod, 0) / recommended.length : 0
    };
  }, [vehicles, recommendations]);

  // Chart data for visualizations
  const chartData = useMemo(() => {
    return recommendations.slice(0, 10).map(r => ({
      vehicle: r.vehicleId,
      score: r.score,
      savings: r.annualSavings,
      emissions: r.annualEmissions
    }));
  }, [recommendations]);

  const priorityData = useMemo(() => {
    const high = recommendations.filter(r => r.priority === 'High').length;
    const medium = recommendations.filter(r => r.priority === 'Medium').length;
    const low = recommendations.filter(r => r.priority === 'Low').length;
    
    const data = [
      { name: 'High', value: high, color: '#f44336', fullName: 'High Priority' },
      { name: 'Medium', value: medium, color: '#ff9800', fullName: 'Medium Priority' },
      { name: 'Low', value: low, color: '#4caf50', fullName: 'Low Priority' }
    ].filter(item => item.value > 0); // Only show segments with values > 0
    
    // If no data, show a default message
    if (data.length === 0) {
      return [{ name: 'No Data', value: 1, color: '#e0e0e0', fullName: 'No Priority Data' }];
    }
    
    return data;
  }, [recommendations]);

  // getPriorityColor was unused – removed to satisfy linter

  const formatCurrency = (amount: number) => `€${amount.toLocaleString(undefined, { maximumFractionDigits: 0 })}`;
  const formatNumber = (num: number, decimals = 1) => num.toLocaleString(undefined, { maximumFractionDigits: decimals });

  // Handler functions for buttons
  const handleGeneratePlan = () => {
    const recommendedVehicles = recommendations.filter(r => r.priority === 'High' || r.priority === 'Medium');
    if (recommendedVehicles.length === 0) {
      setSnackbarMessage('No vehicles recommended for replacement. All vehicles are already optimal or have low priority.');
      setSnackbarOpen(true);
      return;
    }
    setPlanDialogOpen(true);
  };

  const handleCalculateInvestment = () => {
    setInvestmentDialogOpen(true);
  };

  const handleCloseDialogs = () => {
    setPlanDialogOpen(false);
    setInvestmentDialogOpen(false);
  };

  const handleCloseSnackbar = () => {
    setSnackbarOpen(false);
  };

  // Helper function to wrap text
  const wrapText = (doc: jsPDF, text: string, x: number, y: number, maxWidth: number, lineHeight: number = 6) => {
    const words = text.split(' ');
    let line = '';
    let currentY = y;
    
    for (let i = 0; i < words.length; i++) {
      const testLine = line + words[i] + ' ';
      const testWidth = doc.getTextWidth(testLine);
      
      if (testWidth > maxWidth && i > 0) {
        doc.text(line, x, currentY);
        line = words[i] + ' ';
        currentY += lineHeight;
      } else {
        line = testLine;
      }
    }
    doc.text(line, x, currentY);
    return currentY + lineHeight;
  };

  // Export functions
  const handleExportPlan = () => {
    const doc = new jsPDF();
    const pageWidth = doc.internal.pageSize.getWidth();
    const pageHeight = doc.internal.pageSize.getHeight();
    const margin = 20;
    const maxWidth = pageWidth - (margin * 2);
    let yPosition = 20;

    // Title
    doc.setFontSize(20);
    doc.setFont('helvetica', 'bold');
    doc.text('EV Replacement Plan', pageWidth / 2, yPosition, { align: 'center' });
    yPosition += 20;

    // Date and summary
    doc.setFontSize(12);
    doc.setFont('helvetica', 'normal');
    doc.text(`Generated: ${new Date().toLocaleString()}`, 20, yPosition);
    yPosition += 10;
    doc.text(`Total ICE Vehicles: ${summary.totalIceVehicles}`, 20, yPosition);
    yPosition += 10;
    doc.text(`Recommended for Replacement: ${summary.recommendedReplacements}`, 20, yPosition);
    yPosition += 10;
    doc.text(`Total Annual Savings: ${formatCurrency(summary.totalAnnualSavings)}`, 20, yPosition);
    yPosition += 10;
    doc.text(`Total CO₂ Reduction: ${formatNumber(summary.totalCo2Reduction)} kg`, 20, yPosition);
    yPosition += 20;

    // Summary table
    doc.setFont('helvetica', 'bold');
    doc.text('Investment Summary', 20, yPosition);
    yPosition += 10;

    doc.setFont('helvetica', 'normal');
    const tableData = [
      ['Metric', 'Value'],
      ['Vehicles to Replace', summary.recommendedReplacements.toString()],
      ['Average EV Cost', '€35,000'],
      ['Total Investment', formatCurrency(summary.recommendedReplacements * 35000)],
      ['Annual Savings', formatCurrency(summary.totalAnnualSavings)],
      ['Payback Period', `${formatNumber((summary.recommendedReplacements * 35000) / summary.totalAnnualSavings, 1)} years`],
      ['CO₂ Reduction', `${formatNumber(summary.totalCo2Reduction)} kg`]
    ];

    // Simple table
    tableData.forEach((row, index) => {
      if (yPosition > pageHeight - 20) {
        doc.addPage();
        yPosition = 20;
      }
      
      doc.setFont(index === 0 ? 'helvetica' : 'helvetica', index === 0 ? 'bold' : 'normal');
      doc.text(row[0], margin, yPosition);
      doc.text(row[1], margin + 100, yPosition);
      yPosition += 8;
    });

    yPosition += 10;

    // Vehicle recommendations
    doc.setFont('helvetica', 'bold');
    doc.text('Vehicle Recommendations', 20, yPosition);
    yPosition += 10;

    const recommendedVehicles = recommendations.filter(r => r.priority === 'High' || r.priority === 'Medium');
    recommendedVehicles.forEach((rec, index) => {
      if (yPosition > pageHeight - 50) {
        doc.addPage();
        yPosition = 20;
      }

      doc.setFont('helvetica', 'bold');
      doc.text(`${index + 1}. Vehicle ${rec.vehicleId}`, margin, yPosition);
      yPosition += 8;

      doc.setFont('helvetica', 'normal');
      doc.text(`Priority: ${rec.priority}`, margin + 10, yPosition);
      yPosition += 6;
      doc.text(`Score: ${rec.score}`, margin + 10, yPosition);
      yPosition += 6;
      doc.text(`Annual Savings: ${formatCurrency(rec.annualSavings)}`, margin + 10, yPosition);
      yPosition += 6;
      doc.text(`CO₂ Reduction: ${formatNumber(rec.co2Reduction)} kg`, margin + 10, yPosition);
      yPosition += 6;
      doc.text(`Payback Period: ${formatNumber(rec.paybackPeriod, 1)} years`, margin + 10, yPosition);
      yPosition += 6;
      
      // Use text wrapping for reasons
      const reasonsText = `Reasons: ${rec.reasons.join(', ')}`;
      yPosition = wrapText(doc, reasonsText, margin + 10, yPosition, maxWidth - 10);
      yPosition += 10;
    });

    // Footer
    doc.setFontSize(8);
    doc.setFont('helvetica', 'normal');
    doc.text('Generated by Fleet Sustainability Dashboard', pageWidth / 2, pageHeight - 10, { align: 'center' });

    // Save the PDF
    doc.save('ev_replacement_plan.pdf');
    setSnackbarMessage('EV Replacement Plan exported successfully!');
    setSnackbarOpen(true);
  };

  const handleDownloadReport = () => {
    const doc = new jsPDF();
    const pageWidth = doc.internal.pageSize.getWidth();
    const pageHeight = doc.internal.pageSize.getHeight();
    const margin = 20;
    const maxWidth = pageWidth - (margin * 2);
    let yPosition = 20;

    // Title
    doc.setFontSize(20);
    doc.setFont('helvetica', 'bold');
    doc.text('Fleet Electrification Investment Report', pageWidth / 2, yPosition, { align: 'center' });
    yPosition += 20;

    // Executive summary
    doc.setFontSize(14);
    doc.setFont('helvetica', 'bold');
    doc.text('Executive Summary', 20, yPosition);
    yPosition += 10;

    doc.setFontSize(12);
    doc.setFont('helvetica', 'normal');
    const summaryText = `This report analyzes the electrification potential of your fleet and provides detailed investment recommendations based on usage patterns, emissions, and cost savings.`;
    yPosition = wrapText(doc, summaryText, margin, yPosition, maxWidth);
    yPosition += 15;

    // Key metrics
    doc.setFont('helvetica', 'bold');
    doc.text('Key Metrics', 20, yPosition);
    yPosition += 10;

    const metrics = [
      ['Total ICE Vehicles', summary.totalIceVehicles.toString()],
      ['Recommended for EV Replacement', summary.recommendedReplacements.toString()],
      ['Total Investment Required', formatCurrency(summary.recommendedReplacements * 35000)],
      ['Annual Fuel Cost Savings', formatCurrency(summary.totalAnnualSavings * 0.7)],
      ['Annual Maintenance Savings', formatCurrency(summary.totalAnnualSavings * 0.2)],
      ['Total Annual Savings', formatCurrency(summary.totalAnnualSavings)],
      ['Annual CO₂ Reduction', `${formatNumber(summary.totalCo2Reduction)} kg`],
      ['Average Payback Period', `${formatNumber(summary.averagePaybackPeriod, 1)} years`]
    ];

    metrics.forEach(([label, value]) => {
      if (yPosition > pageHeight - 20) {
        doc.addPage();
        yPosition = 20;
      }
      doc.setFont('helvetica', 'normal');
      doc.text(`${label}:`, margin, yPosition);
      doc.text(value, margin + 100, yPosition);
      yPosition += 8;
    });

    yPosition += 15;

    // Financial analysis
    doc.setFont('helvetica', 'bold');
    doc.text('Financial Analysis', 20, yPosition);
    yPosition += 10;

    const totalInvestment = summary.recommendedReplacements * 35000;
    const annualSavings = summary.totalAnnualSavings;
    const paybackYears = totalInvestment / annualSavings;

    doc.setFont('helvetica', 'normal');
    doc.text(`Total Investment: ${formatCurrency(totalInvestment)}`, 20, yPosition);
    yPosition += 6;
    doc.text(`Annual Savings: ${formatCurrency(annualSavings)}`, 20, yPosition);
    yPosition += 6;
    doc.text(`Payback Period: ${formatNumber(paybackYears, 1)} years`, 20, yPosition);
    yPosition += 6;
    doc.text(`5-Year Savings: ${formatCurrency(annualSavings * 5)}`, 20, yPosition);
    yPosition += 6;
    doc.text(`10-Year Savings: ${formatCurrency(annualSavings * 10)}`, 20, yPosition);
    yPosition += 15;

    // Environmental impact
    doc.setFont('helvetica', 'bold');
    doc.text('Environmental Impact', 20, yPosition);
    yPosition += 10;

    doc.setFont('helvetica', 'normal');
    doc.text(`Annual CO₂ Reduction: ${formatNumber(summary.totalCo2Reduction)} kg`, 20, yPosition);
    yPosition += 6;
    doc.text(`5-Year CO₂ Reduction: ${formatNumber(summary.totalCo2Reduction * 5)} kg`, 20, yPosition);
    yPosition += 6;
    doc.text(`10-Year CO₂ Reduction: ${formatNumber(summary.totalCo2Reduction * 10)} kg`, 20, yPosition);
    yPosition += 15;

    // Recommendations
    doc.setFont('helvetica', 'bold');
    doc.text('Detailed Recommendations', 20, yPosition);
    yPosition += 10;

    recommendations.forEach((rec, index) => {
      if (yPosition > pageHeight - 50) {
        doc.addPage();
        yPosition = 20;
      }

      doc.setFont('helvetica', 'bold');
      doc.text(`${index + 1}. Vehicle ${rec.vehicleId}`, margin, yPosition);
      yPosition += 8;

      doc.setFont('helvetica', 'normal');
      doc.text(`Priority: ${rec.priority}`, margin + 10, yPosition);
      yPosition += 6;
      doc.text(`Score: ${rec.score}/100`, margin + 10, yPosition);
      yPosition += 6;
      doc.text(`Annual Savings: ${formatCurrency(rec.annualSavings)}`, margin + 10, yPosition);
      yPosition += 6;
      doc.text(`CO₂ Reduction: ${formatNumber(rec.co2Reduction)} kg/year`, margin + 10, yPosition);
      yPosition += 6;
      doc.text(`Payback Period: ${formatNumber(rec.paybackPeriod, 1)} years`, margin + 10, yPosition);
      yPosition += 6;
      
      // Use text wrapping for reasons
      const reasonsText = `Reasons: ${rec.reasons.join(', ')}`;
      yPosition = wrapText(doc, reasonsText, margin + 10, yPosition, maxWidth - 10);
      yPosition += 10;
    });

    // Footer
    doc.setFontSize(8);
    doc.setFont('helvetica', 'normal');
    doc.text('Generated by Fleet Sustainability Dashboard', pageWidth / 2, pageHeight - 10, { align: 'center' });

    // Save the PDF
    doc.save('fleet_electrification_investment_report.pdf');
    setSnackbarMessage('Investment Report downloaded successfully!');
    setSnackbarOpen(true);
  };

  if (loading) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>Electrification Planning</Typography>
        <LinearProgress />
        <Typography variant="body2" sx={{ mt: 2 }}>Analyzing vehicle data...</Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">{error}</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: { xs: 2, sm: 3 } }}>
      <Typography variant="h4" gutterBottom sx={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: 1,
        fontSize: { xs: '1.5rem', sm: '2rem', md: '2.125rem' }
      }}>
        <ElectricCar color="primary" />
        <Box component="span" sx={{ display: { xs: 'none', sm: 'inline' } }}>
          Electrification Planning
        </Box>
        <Box component="span" sx={{ display: { xs: 'inline', sm: 'none' } }}>
          Electrification
        </Box>
      </Typography>
      
      <Typography variant="body1" color="text.secondary" paragraph sx={{ 
        fontSize: { xs: '0.875rem', sm: '1rem' },
        display: { xs: 'none', sm: 'block' }
      }}>
        Analyze your ICE vehicles and get recommendations for EV replacements based on usage patterns, 
        emissions, and cost savings potential.
      </Typography>

      {/* Summary Cards */}
      <Grid container spacing={{ xs: 2, sm: 3 }} sx={{ mb: { xs: 3, sm: 4 } }}>
        <Grid item xs={6} sm={6} md={3}>
          <Card>
            <CardContent sx={{ p: { xs: 1.5, sm: 2 } }}>
              <Box display="flex" alignItems="center" gap={1}>
                <LocalGasStation color="primary" />
                <Typography variant="h6" sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
                  {summary.totalIceVehicles}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary" sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>
                ICE Vehicles
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={6} sm={6} md={3}>
          <Card>
            <CardContent sx={{ p: { xs: 1.5, sm: 2 } }}>
              <Box display="flex" alignItems="center" gap={1}>
                <TrendingUp color="success" />
                <Typography variant="h6" sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
                  {summary.recommendedReplacements}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary" sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>
                Recommended for EV
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={6} sm={6} md={3}>
          <Card>
            <CardContent sx={{ p: { xs: 1.5, sm: 2 } }}>
              <Box display="flex" alignItems="center" gap={1}>
                <AttachMoney color="success" />
                <Typography variant="h6" sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
                  {formatCurrency(summary.totalAnnualSavings)}
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary" sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>
                Annual Savings
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={6} sm={6} md={3}>
          <Card>
            <CardContent sx={{ p: { xs: 1.5, sm: 2 } }}>
              <Box display="flex" alignItems="center" gap={1}>
                <Park color="success" />
                <Typography variant="h6" sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
                  {formatNumber(summary.totalCo2Reduction)} kg
                </Typography>
              </Box>
              <Typography variant="body2" color="text.secondary" sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' } }}>
                CO₂ Reduction
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Charts */}
      <Grid container spacing={{ xs: 2, sm: 3 }} sx={{ mb: { xs: 3, sm: 4 } }}>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: { xs: 1.5, sm: 2 } }}>
            <Typography variant="h6" gutterBottom sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
              Priority Distribution
            </Typography>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={priorityData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ fullName, value }) => `${fullName}: ${value}`}
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                >
                  {priorityData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <RechartsTooltip 
                  formatter={(value, name, props) => [value, props.payload.fullName]}
                  labelFormatter={(label) => label}
                />
              </PieChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>
        
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: { xs: 1.5, sm: 2 } }}>
            <Typography variant="h6" gutterBottom sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
              Top 10 Vehicles by Score
            </Typography>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={chartData} margin={{ top: 20, right: 30, left: 20, bottom: 60 }}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis 
                  dataKey="vehicle" 
                  angle={-45}
                  textAnchor="end"
                  height={80}
                  fontSize={10}
                  interval={0}
                  tick={{ fontSize: 10 }}
                />
                <YAxis fontSize={12} />
                <RechartsTooltip 
                  formatter={(value, name) => [value, 'Score']}
                  labelFormatter={(label) => `Vehicle: ${label}`}
                />
                <Bar dataKey="score" fill="#1976d2" />
              </BarChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>
      </Grid>

      {/* Recommendations Table */}
      <Paper sx={{ p: { xs: 1.5, sm: 2 } }}>
        <Typography variant="h6" gutterBottom sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }}>
          Vehicle Recommendations
        </Typography>
        
        {recommendations.length === 0 ? (
          <Alert severity="info">
            No ICE vehicles found in your fleet. All vehicles are already electric!
          </Alert>
        ) : (
          <TableContainer sx={{ 
            overflowX: 'auto',
            '&::-webkit-scrollbar': {
              height: '8px',
            },
            '&::-webkit-scrollbar-track': {
              backgroundColor: '#f1f1f1',
              borderRadius: '4px',
            },
            '&::-webkit-scrollbar-thumb': {
              backgroundColor: '#c1c1c1',
              borderRadius: '4px',
            },
          }}>
            <Table sx={{ minWidth: 600 }}>
              <TableHead>
                <TableRow>
                  <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' }, fontWeight: 'bold' }}>
                    Vehicle ID
                  </TableCell>
                  <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' }, fontWeight: 'bold' }}>
                    Priority
                  </TableCell>
                  <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' }, fontWeight: 'bold' }}>
                    Score
                  </TableCell>
                  <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' }, fontWeight: 'bold' }}>
                    Annual Savings
                  </TableCell>
                  <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' }, fontWeight: 'bold' }}>
                    CO₂ Reduction
                  </TableCell>
                  <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' }, fontWeight: 'bold' }}>
                    Payback Period
                  </TableCell>
                  <TableCell sx={{ fontSize: { xs: '0.75rem', sm: '0.875rem' }, fontWeight: 'bold' }}>
                    Reasons
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {recommendations.map((rec) => (
                  <TableRow key={rec.vehicleId}>
                    <TableCell>
                      <Box display="flex" alignItems="center" gap={1}>
                        <LocalGasStation fontSize="small" />
                        {rec.vehicleId}
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Chip 
                        label={rec.priority} 
                        color={rec.priority === 'High' ? 'error' : rec.priority === 'Medium' ? 'warning' : 'success'}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Box display="flex" alignItems="center" gap={1}>
                        <LinearProgress 
                          variant="determinate" 
                          value={rec.score} 
                          sx={{ width: 60, height: 8, borderRadius: 4 }}
                        />
                        <Typography variant="body2">{rec.score}</Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box display="flex" alignItems="center" gap={0.5}>
                        <TrendingUp color="success" fontSize="small" />
                        <Typography variant="body2" color="success.main">
                          {formatCurrency(rec.annualSavings)}
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Box display="flex" alignItems="center" gap={0.5}>
                        <Park color="success" fontSize="small" />
                        <Typography variant="body2" color="success.main">
                          {formatNumber(rec.co2Reduction)} kg
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {rec.paybackPeriod > 0 ? `${formatNumber(rec.paybackPeriod, 1)} years` : 'N/A'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Tooltip title={rec.reasons.join(', ')}>
                        <IconButton size="small">
                          <Info fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </Paper>

      {/* Action Buttons */}
      <Box sx={{ 
        mt: 3, 
        display: 'flex', 
        gap: 2,
        flexDirection: { xs: 'column', sm: 'row' },
        '& > *': {
          flex: { xs: '1', sm: '0 1 auto' }
        }
      }}>
        <Button 
          variant="contained" 
          startIcon={<ElectricCar />}
          disabled={summary.recommendedReplacements === 0}
          onClick={handleGeneratePlan}
          sx={{ fontSize: { xs: '0.875rem', sm: '1rem' } }}
        >
          <Box component="span" sx={{ display: { xs: 'none', sm: 'inline' } }}>
            Generate EV Replacement Plan
          </Box>
          <Box component="span" sx={{ display: { xs: 'inline', sm: 'none' } }}>
            Generate Plan
          </Box>
        </Button>
        <Button 
          variant="outlined" 
          startIcon={<AttachMoney />}
          onClick={handleCalculateInvestment}
          sx={{ fontSize: { xs: '0.875rem', sm: '1rem' } }}
        >
          <Box component="span" sx={{ display: { xs: 'none', sm: 'inline' } }}>
            Calculate Total Investment
          </Box>
          <Box component="span" sx={{ display: { xs: 'inline', sm: 'none' } }}>
            Calculate Investment
          </Box>
        </Button>
      </Box>

      {/* Generate Plan Dialog */}
      <Dialog 
        open={planDialogOpen} 
        onClose={handleCloseDialogs}
        maxWidth="md"
        fullWidth
        PaperProps={{
          sx: { minHeight: '400px' }
        }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <ElectricCar color="primary" />
          EV Replacement Plan
        </DialogTitle>
        <DialogContent>
          <Typography variant="body1" paragraph>
            Based on your fleet analysis, here's the recommended electrification plan:
          </Typography>
          
          <Grid container spacing={2} sx={{ mb: 3 }}>
            <Grid item xs={12} sm={6}>
              <Card variant="outlined">
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="h6" color="primary">
                    {summary.recommendedReplacements}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Vehicles to Replace
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Card variant="outlined">
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="h6" color="success.main">
                    {formatCurrency(summary.totalAnnualSavings)}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Annual Savings
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Card variant="outlined">
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="h6" color="success.main">
                    {formatNumber(summary.totalCo2Reduction)} kg
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    CO₂ Reduction
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} sm={6}>
              <Card variant="outlined">
                <CardContent sx={{ p: 2 }}>
                  <Typography variant="h6" color="info.main">
                    {formatNumber(summary.averagePaybackPeriod, 1)} years
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Average Payback Period
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          <Typography variant="h6" gutterBottom>
            Recommended Replacements
          </Typography>
          <List>
            {recommendations
              .filter(r => r.priority === 'High' || r.priority === 'Medium')
              .map((rec, index) => (
                <React.Fragment key={rec.vehicleId}>
                  <ListItem sx={{ px: 0 }}>
                    <ListItemText
                      primary={
                        <Box display="flex" alignItems="center" gap={1}>
                          <Typography variant="subtitle1">
                            {rec.vehicleId}
                          </Typography>
                          <Chip 
                            label={rec.priority} 
                            color={rec.priority === 'High' ? 'error' : 'warning'}
                            size="small"
                          />
                        </Box>
                      }
                      secondary={
                        <Box>
                          <Typography variant="body2" color="text.secondary">
                            Annual Savings: {formatCurrency(rec.annualSavings)} • 
                            CO₂ Reduction: {formatNumber(rec.co2Reduction)} kg • 
                            Payback: {formatNumber(rec.paybackPeriod, 1)} years
                          </Typography>
                          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                            Reasons: {rec.reasons.join(', ')}
                          </Typography>
                        </Box>
                      }
                    />
                  </ListItem>
                  {index < recommendations.filter(r => r.priority === 'High' || r.priority === 'Medium').length - 1 && (
                    <Divider />
                  )}
                </React.Fragment>
              ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialogs}>Close</Button>
          <Button variant="contained" onClick={handleExportPlan}>
            Export Plan
          </Button>
        </DialogActions>
      </Dialog>

      {/* Investment Calculation Dialog */}
      <Dialog 
        open={investmentDialogOpen} 
        onClose={handleCloseDialogs}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <AttachMoney color="primary" />
          Total Investment Calculation
        </DialogTitle>
        <DialogContent>
          <Typography variant="body1" paragraph>
            Investment breakdown for electrifying your recommended vehicles:
          </Typography>
          
          <Box sx={{ mb: 3 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <Card variant="outlined">
                  <CardContent>
                    <Typography variant="h6" color="primary" gutterBottom>
                      Investment Summary
                    </Typography>
                    <Box display="flex" justifyContent="space-between" mb={1}>
                      <Typography variant="body2">Vehicles to Replace:</Typography>
                      <Typography variant="body2" fontWeight="bold">
                        {summary.recommendedReplacements}
                      </Typography>
                    </Box>
                    <Box display="flex" justifyContent="space-between" mb={1}>
                      <Typography variant="body2">Average EV Cost:</Typography>
                      <Typography variant="body2" fontWeight="bold">
                        {formatCurrency(35000)}
                      </Typography>
                    </Box>
                    <Divider sx={{ my: 1 }} />
                    <Box display="flex" justifyContent="space-between" mb={2}>
                      <Typography variant="h6">Total Investment:</Typography>
                      <Typography variant="h6" color="primary">
                        {formatCurrency(summary.recommendedReplacements * 35000)}
                      </Typography>
                    </Box>
                    <Box display="flex" justifyContent="space-between" mb={1}>
                      <Typography variant="body2">Annual Savings:</Typography>
                      <Typography variant="body2" color="success.main" fontWeight="bold">
                        {formatCurrency(summary.totalAnnualSavings)}
                      </Typography>
                    </Box>
                    <Box display="flex" justifyContent="space-between">
                      <Typography variant="body2">Payback Period:</Typography>
                      <Typography variant="body2" fontWeight="bold">
                        {formatNumber((summary.recommendedReplacements * 35000) / summary.totalAnnualSavings, 1)} years
                      </Typography>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>
          </Box>

          <Alert severity="info" sx={{ mb: 2 }}>
            <Typography variant="body2">
              <strong>Note:</strong> This calculation assumes an average EV cost of €35,000 per vehicle. 
              Actual costs may vary based on vehicle specifications, government incentives, and bulk purchasing discounts.
            </Typography>
          </Alert>

          <Typography variant="h6" gutterBottom>
            Financial Benefits
          </Typography>
          <List dense>
            <ListItem>
              <ListItemText
                primary="Fuel Cost Savings"
                secondary={`€${(summary.totalAnnualSavings * 0.7).toLocaleString()} per year`}
              />
            </ListItem>
            <ListItem>
              <ListItemText
                primary="Maintenance Savings"
                secondary={`€${(summary.totalAnnualSavings * 0.2).toLocaleString()} per year`}
              />
            </ListItem>
            <ListItem>
              <ListItemText
                primary="Environmental Benefits"
                secondary={`${formatNumber(summary.totalCo2Reduction)} kg CO₂ reduction annually`}
              />
            </ListItem>
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialogs}>Close</Button>
          <Button variant="contained" onClick={handleDownloadReport}>
            Download Report
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={handleCloseSnackbar} severity="info" sx={{ width: '100%' }}>
          {snackbarMessage}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default ElectrificationPlanning;
