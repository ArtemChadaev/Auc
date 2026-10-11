import type {
  UploadUrlRequest,
  UploadUrlResponse,
  ConfirmUploadResponse,
  UploadStatusResponse,
} from '~/api/types';

// TODO(real-api): POST /api/upload/url
export async function getUploadUrlReal(_req: UploadUrlRequest): Promise<UploadUrlResponse> {
  throw new Error('Real API: getUploadUrl is not implemented');
}

// TODO(real-api): PUT <presigned-url>
export async function uploadBinaryToS3Real(
  _url: string,
  _file: File | Blob,
  _onProgress?: (percent: number) => void
): Promise<void> {
  throw new Error('Real API: uploadBinaryToS3 is not implemented');
}

// TODO(real-api): POST /api/upload/confirm
export async function confirmUploadReal(_key: string): Promise<ConfirmUploadResponse> {
  throw new Error('Real API: confirmUpload is not implemented');
}

// TODO(real-api): GET /api/upload/status?key=<uuid>
export async function getUploadStatusReal(_key: string): Promise<UploadStatusResponse> {
  throw new Error('Real API: getUploadStatus is not implemented');
}

// TODO(real-api): POST /api/upload/cancel
export async function cancelUploadReal(_key?: string): Promise<void> {
  throw new Error('Real API: cancelUpload is not implemented');
}
