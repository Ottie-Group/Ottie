import React, { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import jsQR from 'jsqr';
import * as OTPAuth from 'otpauth';
import { useVaultStore } from '../store/useVaultStore';
import { useToastStore } from '../store/useToastStore';
import { BrandHeader } from '../components/common/BrandHeader';
import { Button } from '../components/common/Button';
import { FormGroup, Label, Input, HelpText } from '../components/common/Input';
import { CustomSelect } from '../components/common/CustomSelect';
import { AppWrapper, FormCard, FormHeader } from '../components/layout/AppWrapper';
import { Styled } from './AddTokenPage.Styled';

const CATEGORY_OPTIONS = [
  { value: 'Personal', label: 'Personal' },
  { value: 'Work', label: 'Work' },
  { value: 'Finance', label: 'Finance' },
  { value: 'Cloud / Dev', label: 'Cloud / Dev' },
  { value: 'Gaming', label: 'Gaming' },
  { value: 'custom', label: 'Custom...' },
];

export function AddTokenPage() {
  const navigate = useNavigate();
  const [tab, setTab] = useState<'manual' | 'qr'>('manual');
  const [secret, setSecret] = useState('');
  const [issuer, setIssuer] = useState('');
  const [accountName, setAccountName] = useState('');
  const [categoryChoice, setCategoryChoice] = useState('Personal');
  const [customCategory, setCustomCategory] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isDragging, setIsDragging] = useState(false);

  // Time ticker state for calculating live TOTP
  const [secondsRemaining, setSecondsRemaining] = useState<number>(30);

  // Camera scanner state
  const [isCameraActive, setIsCameraActive] = useState(false);
  const [cameraError, setCameraError] = useState<string | null>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const animationFrameRef = useRef<number | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const cameraInputRef = useRef<HTMLInputElement>(null);
  const customInputRef = useRef<HTMLInputElement>(null);

  const addToken = useVaultStore((state) => state.addToken);
  const showToast = useToastStore((state) => state.showToast);

  // Helper to extract clean Base32 secret from either a raw string or an otpauth:// URI
  const extractSecretAndMetadata = useCallback((input: string) => {
    const trimmed = input.trim();
    if (trimmed.toLowerCase().startsWith('otpauth://')) {
      try {
        const parsed = new URL(trimmed);
        if (parsed.protocol === 'otpauth:') {
          const sec = parsed.searchParams.get('secret') || '';
          const iss = parsed.searchParams.get('issuer') || '';
          const label = decodeURIComponent(parsed.pathname.replace(/^\/\/?(totp|hotp)\//i, ''));
          let acc = '';
          let derivedIssuer = iss;
          if (label.includes(':')) {
            const parts = label.split(':');
            if (!derivedIssuer) derivedIssuer = parts[0].trim();
            acc = parts[1].trim();
          } else if (label) {
            acc = label.trim();
          }
          return {
            secret: sec.replace(/\s+/g, '').toUpperCase(),
            issuer: derivedIssuer,
            accountName: acc,
            isUri: true,
            isValidUri: Boolean(sec),
          };
        }
      } catch {
        return { secret: '', issuer: '', accountName: '', isUri: true, isValidUri: false };
      }
    }
    return {
      secret: trimmed.replace(/\s+/g, '').toUpperCase(),
      issuer: '',
      accountName: '',
      isUri: false,
      isValidUri: false,
    };
  }, []);

  // Tick every second to update live TOTP preview
  useEffect(() => {
    const timer = setInterval(() => {
      setSecondsRemaining(30 - (Math.floor(Date.now() / 1000) % 30));
    }, 1000);
    return () => clearInterval(timer);
  }, []);

  // Compute live TOTP code directly from secret and current timestamp
  let livePreviewCode: string | null = null;
  if (secret.trim()) {
    try {
      const extracted = extractSecretAndMetadata(secret);
      if (extracted.secret && /^[A-Z2-7=]+$/.test(extracted.secret) && extracted.secret.length >= 8) {
        const totp = new OTPAuth.TOTP({
          secret: extracted.secret,
          algorithm: 'SHA1',
          digits: 6,
          period: 30,
        });
        livePreviewCode = totp.generate();
      }
    } catch {
      livePreviewCode = null;
    }
  }

  const handleCategoryChange = (val: string) => {
    setCategoryChoice(val);
    if (val === 'custom') {
      setTimeout(() => customInputRef.current?.focus(), 50);
    }
  };

  const parseOtpAuthUri = useCallback((data: string) => {
    const trimmed = data.trim();
    if (trimmed.toLowerCase().startsWith('otpauth://')) {
      try {
        const parsed = new URL(trimmed);
        if (parsed.protocol === 'otpauth:') {
          const sec = parsed.searchParams.get('secret');
          const iss = parsed.searchParams.get('issuer');
          const label = decodeURIComponent(parsed.pathname.replace(/^\/\/?(totp|hotp)\//i, ''));

          if (label.includes(':')) {
            const parts = label.split(':');
            if (!iss && parts[0].trim()) setIssuer(parts[0].trim());
            if (parts[1].trim()) setAccountName(parts[1].trim());
          } else if (label.trim()) {
            setAccountName(label.trim());
          }
          if (sec) setSecret(sec.trim());
          if (iss && iss.trim()) setIssuer(iss.trim());
          showToast('Scanned QR code successfully!');
          return;
        }
      } catch (_e) {
        // Not a valid URL, fallback
      }
    }

    const clean = trimmed.replace(/\s+/g, '').toUpperCase();
    if (/^[A-Z2-7=]+$/.test(clean) && clean.length >= 8) {
      setSecret(clean);
      showToast('Captured secret key from QR code!');
    } else {
      showToast('Scanned QR code data could not be parsed.');
    }
  }, [showToast]);

  const handleSecretChange = (val: string) => {
    setSecret(val);
    if (val.toLowerCase().startsWith('otpauth://')) {
      parseOtpAuthUri(val);
    }
  };

  // Robust multi-pass QR code decoder from image
  const decodeQRFromImage = useCallback(async (img: HTMLImageElement): Promise<string | null> => {
    // Pass 1: Native BarcodeDetector (Chrome, Edge, Safari 17+, Android)
    if ('BarcodeDetector' in window) {
      try {
        const detector = new (window as any).BarcodeDetector({ formats: ['qr_code'] });
        const barcodes = await detector.detect(img);
        if (barcodes && barcodes.length > 0 && barcodes[0].rawValue) {
          return barcodes[0].rawValue;
        }
      } catch (_e) {
        // Fallback to jsQR
      }
    }

    const naturalWidth = img.naturalWidth || img.width;
    const naturalHeight = img.naturalHeight || img.height;
    if (!naturalWidth || !naturalHeight) return null;

    const maxDim = Math.max(naturalWidth, naturalHeight);
    const scales = [1.0, 1600 / maxDim, 1200 / maxDim, 800 / maxDim, 500 / maxDim];
    const uniqueScales = Array.from(new Set(scales.map((s) => Math.min(1.0, s)))).filter((s) => s > 0);

    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d', { willReadFrequently: true });
    if (!ctx) return null;

    // Pass 2: Multi-resolution scaling
    for (const scale of uniqueScales) {
      const targetW = Math.round(naturalWidth * scale);
      const targetH = Math.round(naturalHeight * scale);
      canvas.width = targetW;
      canvas.height = targetH;
      ctx.drawImage(img, 0, 0, targetW, targetH);

      const imgData = ctx.getImageData(0, 0, targetW, targetH);
      const code = jsQR(imgData.data, targetW, targetH, {
        inversionAttempts: 'attemptBoth',
      });
      if (code && code.data) {
        return code.data;
      }
    }

    // Pass 3: Center cropped region (for full screenshot photos)
    if (naturalWidth > 300 && naturalHeight > 300) {
      for (const cropRatio of [0.75, 0.5]) {
        const cropW = Math.round(naturalWidth * cropRatio);
        const cropH = Math.round(naturalHeight * cropRatio);
        const startX = Math.round((naturalWidth - cropW) / 2);
        const startY = Math.round((naturalHeight - cropH) / 2);

        canvas.width = cropW;
        canvas.height = cropH;
        ctx.drawImage(img, startX, startY, cropW, cropH, 0, 0, cropW, cropH);

        const imgData = ctx.getImageData(0, 0, cropW, cropH);
        const code = jsQR(imgData.data, cropW, cropH, {
          inversionAttempts: 'attemptBoth',
        });
        if (code && code.data) {
          return code.data;
        }
      }
    }

    return null;
  }, []);

  const handleFile = useCallback((file: File) => {
    if (!file.type.startsWith('image/')) {
      showToast('Please select a valid image file.');
      return;
    }

    const reader = new FileReader();
    reader.onload = (e) => {
      const img = new Image();
      img.onload = async () => {
        try {
          const result = await decodeQRFromImage(img);
          if (result) {
            parseOtpAuthUri(result);
          } else {
            showToast('No QR code detected. Please ensure the QR code is clearly visible in the image.');
          }
        } catch (_err) {
          showToast('Failed to process image for QR code.');
        }
      };
      img.onerror = () => {
        showToast('Could not load image file.');
      };
      img.src = e.target?.result as string;
    };
    reader.readAsDataURL(file);
  }, [decodeQRFromImage, parseOtpAuthUri, showToast]);

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFile(e.dataTransfer.files[0]);
    }
  };

  const stopCamera = useCallback(() => {
    if (animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }
    setIsCameraActive(false);
  }, []);

  // Camera video stream and scanning loop lifecycle
  useEffect(() => {
    if (!isCameraActive) return;

    let activeStream: MediaStream | null = null;
    let animFrame: number | null = null;
    let isCancelled = false;

    async function initCamera() {
      setCameraError(null);
      try {
        if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
          setCameraError('Live camera video streaming requires HTTPS or localhost. Over plain HTTP on a local network, please use "Take / Pick Photo" or upload an image below.');
          setIsCameraActive(false);
          return;
        }

        const stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: { ideal: 'environment' }, width: { ideal: 1280 }, height: { ideal: 720 } },
        });

        if (isCancelled) {
          stream.getTracks().forEach((t) => t.stop());
          return;
        }

        activeStream = stream;
        streamRef.current = stream;

        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          videoRef.current.setAttribute('playsinline', 'true');
          await videoRef.current.play();

          const canvas = document.createElement('canvas');
          const ctx = canvas.getContext('2d', { willReadFrequently: true });

          let detector: any = null;
          if ('BarcodeDetector' in window) {
            try {
              detector = new (window as any).BarcodeDetector({ formats: ['qr_code'] });
            } catch (_e) {
              detector = null;
            }
          }

          const scanLoop = async () => {
            if (isCancelled || !videoRef.current) return;
            const video = videoRef.current;

            if (video.readyState >= 2) {
              if (detector) {
                try {
                  const barcodes = await detector.detect(video);
                  if (barcodes && barcodes.length > 0 && barcodes[0].rawValue) {
                    stopCamera();
                    parseOtpAuthUri(barcodes[0].rawValue);
                    return;
                  }
                } catch (_e) {
                  // Fallback to canvas jsQR
                }
              }

              if (ctx && video.videoWidth > 0 && video.videoHeight > 0) {
                canvas.width = video.videoWidth;
                canvas.height = video.videoHeight;
                ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
                const imgData = ctx.getImageData(0, 0, canvas.width, canvas.height);
                const qr = jsQR(imgData.data, canvas.width, canvas.height, {
                  inversionAttempts: 'attemptBoth',
                });
                if (qr && qr.data) {
                  stopCamera();
                  parseOtpAuthUri(qr.data);
                  return;
                }
              }
            }

            animFrame = requestAnimationFrame(scanLoop);
          };

          animFrame = requestAnimationFrame(scanLoop);
        }
      } catch (_err: any) {
        setCameraError('Camera access unavailable or denied. Please check browser permissions or upload an image.');
        setIsCameraActive(false);
      }
    }

    initCamera();

    return () => {
      isCancelled = true;
      if (animFrame) cancelAnimationFrame(animFrame);
      if (activeStream) {
        activeStream.getTracks().forEach((t) => t.stop());
      }
    };
  }, [isCameraActive, parseOtpAuthUri, stopCamera]);

  const startCamera = () => {
    setCameraError(null);
    if (typeof navigator === 'undefined' || !navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      setCameraError('Live camera video streaming requires HTTPS or localhost. Over plain HTTP on a local network, please use "Take / Pick Photo" or upload an image below.');
      return;
    }
    setIsCameraActive(true);
  };

  // Listen for clipboard image paste on QR tab
  useEffect(() => {
    const handlePaste = (e: ClipboardEvent) => {
      if (tab !== 'qr') return;
      if (e.clipboardData && e.clipboardData.items) {
        for (let i = 0; i < e.clipboardData.items.length; i++) {
          const item = e.clipboardData.items[i];
          if (item.type.startsWith('image/')) {
            const blob = item.getAsFile();
            if (blob) {
              handleFile(blob);
              showToast('Processing pasted QR screenshot...');
            }
          }
        }
      }
    };

    window.addEventListener('paste', handlePaste);
    return () => window.removeEventListener('paste', handlePaste);
  }, [tab, handleFile, showToast]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const extracted = extractSecretAndMetadata(secret);

    if (extracted.isUri && !extracted.isValidUri) {
      showToast('Invalid otpauth URI. Make sure it includes a valid "secret=" parameter.');
      return;
    }

    if (!extracted.secret) {
      showToast('Please enter a secret key or otpauth URI.');
      return;
    }

    if (!/^[A-Z2-7=]+$/.test(extracted.secret)) {
      showToast('Secret key must be a valid Base32 string (letters A-Z and digits 2-7).');
      return;
    }

    if (extracted.secret.length < 8) {
      showToast('Secret key must be at least 8 characters long.');
      return;
    }

    const finalIssuer = (issuer.trim() || extracted.issuer).trim();
    if (!finalIssuer) {
      showToast('Please enter a service/issuer name (e.g. GitHub).');
      return;
    }

    if (finalIssuer.length > 100) {
      showToast('Issuer name must not exceed 100 characters.');
      return;
    }

    const finalAccount = (accountName.trim() || extracted.accountName).trim();
    if (finalAccount.length > 100) {
      showToast('Account name must not exceed 100 characters.');
      return;
    }

    if (categoryChoice === 'custom' && !customCategory.trim()) {
      showToast('Please enter a label for your custom category.');
      return;
    }

    const finalCategory =
      categoryChoice === 'custom'
        ? customCategory.trim()
        : categoryChoice;

    try {
      setIsSubmitting(true);
      await addToken({
        secret: extracted.secret,
        issuer: finalIssuer,
        accountName: finalAccount,
        category: finalCategory,
      });
      showToast(`Pebble for ${finalIssuer} added!`);
      navigate('/', { replace: true });
    } catch (err: any) {
      showToast(err.message || 'Failed to add pebble.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <AppWrapper size="default">
      <BrandHeader />

      <Styled.BackNav>
        <Styled.BackBtn onClick={() => navigate('/')}>
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <line x1="19" y1="12" x2="5" y2="12" />
            <polyline points="12 19 5 12 12 5" />
          </svg>
          Back to Dashboard
        </Styled.BackBtn>
      </Styled.BackNav>

      <FormCard>
        <FormHeader>
          <h2>Add New Secret Pebble</h2>
          <p>Encrypt a new 2FA pebble directly in your private den.</p>
        </FormHeader>

        <Styled.TabsWrap>
          <Styled.TabBtn
            type="button"
            isActive={tab === 'manual'}
            onClick={() => {
              stopCamera();
              setTab('manual');
            }}
          >
            Manual Entry
          </Styled.TabBtn>
          <Styled.TabBtn
            type="button"
            isActive={tab === 'qr'}
            onClick={() => setTab('qr')}
          >
            Scan / Upload QR
          </Styled.TabBtn>
        </Styled.TabsWrap>

        {tab === 'qr' && (
          <div>
            {/* Live Camera Scanner */}
            {isCameraActive ? (
              <Styled.CameraContainer>
                <Styled.VideoStream ref={videoRef} autoPlay playsInline muted />
                <Styled.ScanOverlay>
                  <Styled.ScanBox />
                </Styled.ScanOverlay>
                <Styled.CameraControls style={{ padding: '12px', background: '#0f172a' }}>
                  <Button variant="danger" size="sm" onClick={stopCamera}>
                    Stop Camera
                  </Button>
                </Styled.CameraControls>
              </Styled.CameraContainer>
            ) : (
              <div style={{ display: 'flex', gap: '10px', marginBottom: '16px' }}>
                <Button
                  type="button"
                  variant="outline"
                  fullWidth
                  onClick={startCamera}
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" style={{ marginRight: '6px' }}>
                    <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
                    <circle cx="12" cy="13" r="4" />
                  </svg>
                  Scan with Camera
                </Button>

                <Button
                  type="button"
                  variant="outline"
                  fullWidth
                  onClick={() => cameraInputRef.current?.click()}
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" style={{ marginRight: '6px' }}>
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="17 8 12 3 7 8" />
                    <line x1="12" y1="3" x2="12" y2="15" />
                  </svg>
                  Take / Pick Photo
                </Button>
              </div>
            )}

            {cameraError && (
              <p style={{ fontSize: '12px', color: '#b91c1c', marginBottom: '14px', textAlign: 'center' }}>
                {cameraError}
              </p>
            )}

            {/* Desktop Drag-and-Drop & Clipboard Paste Area */}
            <Styled.DropZone
              isDragging={isDragging}
              onDragOver={(e) => {
                e.preventDefault();
                setIsDragging(true);
              }}
              onDragLeave={() => setIsDragging(false)}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
            >
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#059669" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" style={{ margin: '0 auto 10px' }}>
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
              <h4 style={{ fontSize: '14px', fontWeight: 700, color: '#0f172a', marginBottom: '4px' }}>
                Drop QR screenshot here, click to browse, or paste (Ctrl+V)
              </h4>
              <p style={{ fontSize: '12px', color: '#64748b' }}>
                Supports PNG, JPEG, SVG, WEBP QR codes
              </p>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                style={{ display: 'none' }}
                onChange={(e) => {
                  if (e.target.files && e.target.files[0]) {
                    handleFile(e.target.files[0]);
                  }
                }}
              />
              <input
                ref={cameraInputRef}
                type="file"
                accept="image/*"
                capture="environment"
                style={{ display: 'none' }}
                onChange={(e) => {
                  if (e.target.files && e.target.files[0]) {
                    handleFile(e.target.files[0]);
                  }
                }}
              />
            </Styled.DropZone>
          </div>
        )}

        {tab === 'qr' && secret && (
          <div style={{ background: '#f0fdf4', border: '1.5px solid #86efac', borderRadius: '12px', padding: '12px 16px', marginBottom: '20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <div>
              <div style={{ color: '#166534', fontWeight: 800, fontSize: '13px' }}>✓ QR Secret Captured</div>
              <div style={{ fontSize: '12px', color: '#15803d', marginTop: '2px' }}>{issuer || 'Service'} pebble ready to save</div>
            </div>
            <Button type="button" variant="outline" size="sm" onClick={() => { setSecret(''); setIssuer(''); setAccountName(''); }}>
              Scan Another
            </Button>
          </div>
        )}

        <form onSubmit={handleSubmit}>
          {tab === 'manual' && (
            <FormGroup>
              <Label htmlFor="secret">Secret Key (Base32) or otpauth URI *</Label>
              <Input
                id="secret"
                type="text"
                placeholder="e.g. JBSWY3DPEHPK3PXP"
                value={secret}
                onChange={(e) => handleSecretChange(e.target.value)}
                required
              />
              <HelpText>Paste raw Base32 secret or standard otpauth:// totp uri.</HelpText>
            </FormGroup>
          )}

          <FormGroup>
            <Label htmlFor="issuer">Service / Issuer *</Label>
            <Input
              id="issuer"
              type="text"
              placeholder="e.g. GitHub, AWS, Google, Discord"
              value={issuer}
              onChange={(e) => setIssuer(e.target.value)}
              required
            />
          </FormGroup>

          <FormGroup>
            <Label htmlFor="accountName">Account Name / Email</Label>
            <Input
              id="accountName"
              type="text"
              placeholder="e.g. otter@example.com"
              value={accountName}
              onChange={(e) => setAccountName(e.target.value)}
            />
          </FormGroup>

          <FormGroup>
            <Label htmlFor="category">Category</Label>
            <CustomSelect
              options={CATEGORY_OPTIONS}
              value={categoryChoice}
              onChange={handleCategoryChange}
            />
          </FormGroup>

          {categoryChoice === 'custom' && (
            <FormGroup>
              <Label htmlFor="customCategory">Custom Category Name</Label>
              <Input
                ref={customInputRef}
                id="customCategory"
                type="text"
                placeholder="e.g. Side Projects, Crypto"
                value={customCategory}
                onChange={(e) => setCustomCategory(e.target.value)}
                required
              />
            </FormGroup>
          )}

          {/* Prominent Live Verification Passcode for Service Setup */}
          {livePreviewCode && (
            <Styled.LivePreviewCard style={{ marginTop: '8px', marginBottom: '22px' }}>
              <Styled.LivePreviewHeader>
                <Styled.LiveBadge>
                  <svg width="10" height="10" viewBox="0 0 100 100" fill="none">
                    <circle cx="50" cy="50" r="45" fill="#22c55e" />
                  </svg>
                  Setup Verification Code
                </Styled.LiveBadge>
                <span style={{ fontSize: '12px', fontWeight: 700, color: '#166534' }}>
                  {secondsRemaining}s remaining
                </span>
              </Styled.LivePreviewHeader>
              <Styled.LivePreviewDigits>
                {livePreviewCode.slice(0, 3)} {livePreviewCode.slice(3)}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    if (livePreviewCode) {
                      navigator.clipboard.writeText(livePreviewCode);
                      showToast(`Copied verification code ${livePreviewCode} to clipboard!`);
                    }
                  }}
                  title="Copy verification code"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" style={{ marginRight: '4px' }}>
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                  </svg>
                  Copy Code
                </Button>
              </Styled.LivePreviewDigits>
              <p style={{ fontSize: '12px', color: '#15803d', marginTop: '8px', lineHeight: 1.4, margin: '8px 0 0' }}>
                Enter this 6-digit code into <strong>{issuer.trim() || 'your service'}</strong> to complete 2FA setup on their end.
              </p>
            </Styled.LivePreviewCard>
          )}

          <Button type="submit" variant="primary" fullWidth disabled={isSubmitting}>
            {isSubmitting ? 'Securing Pebble in Den...' : 'Save Pebble to Den'}
          </Button>
        </form>
      </FormCard>
    </AppWrapper>
  );
}
