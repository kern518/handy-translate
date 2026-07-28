import React, { useEffect, useState, useRef } from 'react';
import toast, { Toaster } from 'react-hot-toast';
import { useConfig, useToastStyle, useVoice } from '../../hooks';
import { atom, useAtom } from 'jotai';
import { CaptureSelectedScreen } from '../../../bindings/handy-translate/internal/app/binding';
import { Events, Window } from "@wailsio/runtime";



export default function Screenshot() {
    const [imgurl, setImgurl] = useState('');
    const [isMoved, setIsMoved] = useState(false);
    const [isDown, setIsDown] = useState(false);
    const [mouseDownX, setMouseDownX] = useState(0);
    const [mouseDownY, setMouseDownY] = useState(0);
    const [mouseMoveX, setMouseMoveX] = useState(0);
    const [mouseMoveY, setMouseMoveY] = useState(0);
    const toastStyle = useToastStyle();
    const imgRef = useRef();
    const canvasRef = useRef(null);

    useEffect(() => {
        const unsubscribeScreenshot = Events.On("screenshotBase64", function (result) {
            let base64 = result.data
            setImgurl("data:image/png;base64," + base64)
        })

        return () => {
            if (unsubscribeScreenshot) unsubscribeScreenshot()
        }
    }, [])

    const captureScreenshot = (x, y, width, height) => {
        // const canvas = canvasRef.current;
        // const context = canvas.getContext('2d');

        // // // 创建一个新的 Image 对象
        // const image = new Image();
        // image.src = imgurl; // 替换为您的图片 URL
        // // // 设置截图的起始坐标和截图的宽度和高度

        // // // 在 Canvas 上绘制截图
        // canvas.width = width;
        // canvas.height = height;
        // context.drawImage(image, x, y, width, height, 0, 0, width, height);
        // const base64Data = canvas.toDataURL('image/png');
        // EventsEmit("screenshotCapture", base64Data)
        CaptureSelectedScreen(x, y, x + width, y + height).then(() => {
        })
        setImgurl("")
    }

    return (
        <>
            <img
                ref={imgRef}
                className='fixed top-0 left-0 w-full select-none'
                src={imgurl}
                draggable={false}
                onLoad={() => {
                    if (imgurl !== '' && imgRef.current.complete) {
                        Window.Show()
                        Window.Fullscreen()
                        Window.SetAlwaysOnTop(true)
                    }
                }}
            />
            <div
                className={`fixed bg-[#2080f020] border border-solid border-sky-500 ${!isMoved && 'hidden'}`}
                style={{
                    top: Math.min(mouseDownY, mouseMoveY),
                    left: Math.min(mouseDownX, mouseMoveX),
                    bottom: screen.height - Math.max(mouseDownY, mouseMoveY),
                    right: screen.width - Math.max(mouseDownX, mouseMoveX),
                }}
            />
            <div
                className='fixed top-0 left-0 bottom-0 right-0 cursor-crosshair select-none'
                onMouseDown={(e) => {
                    if (e.button === 0) {
                        setIsDown(true);
                        setMouseDownX(e.clientX);
                        setMouseDownY(e.clientY);
                    } else {
                        Window.Hide()
                    }
                }}
                onMouseMove={(e) => {
                    if (isDown) {
                        setIsMoved(true);
                        setMouseMoveX(e.clientX);
                        setMouseMoveY(e.clientY);
                    }
                }}
                onMouseUp={async (e) => {
                    setIsDown(false);
                    setIsMoved(false);
                    if (e.button === 0) {
                        const imgWidth = imgRef.current.naturalWidth;
                        const dpi = imgWidth / screen.width;
                        const left = Math.floor(Math.min(mouseDownX, e.clientX) * dpi);
                        const top = Math.floor(Math.min(mouseDownY, e.clientY) * dpi);
                        const right = Math.floor(Math.max(mouseDownX, e.clientX) * dpi);
                        const bottom = Math.floor(Math.max(mouseDownY, e.clientY) * dpi);
                        const width = right - left;
                        const height = bottom - top;
                        if (width <= 0 || height <= 0) {
                            toast.error('Screenshot area is too small', { style: toastStyle });
                        } else {
                            captureScreenshot(left, top, width, height)
                        }
                    }
                    Window.Hide()
                }}
            />
        </>
    );
}
