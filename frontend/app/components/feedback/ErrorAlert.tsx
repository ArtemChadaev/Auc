import React from 'react';
import { AlertCircle, AlertTriangle, Info } from 'lucide-react';
import { getErrorMessage } from '~/lib/error-messages';
import { isApiError } from '~/api/errors';

interface ErrorAlertProps {
  error: unknown;
  className?: string;
  onDismiss?: () => void;
}

export function ErrorAlert({ error, className = '', onDismiss }: ErrorAlertProps) {
  if (!error) return null;

  let code = 'unknown_error';
  let fallbackMessage = '';

  if (isApiError(error)) {
    code = error.code;
    fallbackMessage = error.message;
  } else if (error instanceof Error) {
    code = 'system_error';
    fallbackMessage = error.message;
  } else if (typeof error === 'string') {
    fallbackMessage = error;
  }

  const descriptor = getErrorMessage(code, fallbackMessage);

  const isWarning = descriptor.severity === 'warning';
  const isInfo = descriptor.severity === 'info';

  const bgBorder = isInfo
    ? 'bg-blue-50 border-blue-200 text-blue-900'
    : isWarning
    ? 'bg-amber-50 border-amber-200 text-amber-900'
    : 'bg-rose-50 border-rose-200 text-rose-900';

  const Icon = isInfo ? Info : isWarning ? AlertTriangle : AlertCircle;

  return (
    <div
      role="alert"
      className={`flex items-start gap-3 p-4 rounded-xl border text-sm transition-all ${bgBorder} ${className}`}
    >
      <Icon className="w-5 h-5 shrink-0 mt-0.5" />
      <div className="flex-1">
        <h4 className="font-semibold">{descriptor.title}</h4>
        <p className="mt-0.5 opacity-90 leading-relaxed">{descriptor.description}</p>
      </div>
      {onDismiss && (
        <button
          onClick={onDismiss}
          className="text-xs font-medium opacity-60 hover:opacity-100 transition-opacity ml-2"
        >
          ✕
        </button>
      )}
    </div>
  );
}
