"use client";

import { useEffect, useState } from "react";
import QRCode from "qrcode";

interface QRCodeProps {
  value: string;
  size?: number;
  alt?: string;
  className?: string;
}

export default function QRCodeComponent({
  value,
  size = 128,
  alt = "QR code",
  className = "",
}: QRCodeProps) {
  const [dataUrl, setDataUrl] = useState<string>("");

  useEffect(() => {
    QRCode.toDataURL(value, { width: size, margin: 1 })
      .then(setDataUrl)
      .catch(console.error);
  }, [value, size]);

  return <img src={dataUrl} alt={alt} width={size} height={size} className={className} />;
}