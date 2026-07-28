import { Divider, Button, Popover, PopoverTrigger, PopoverContent, Tooltip } from '@nextui-org/react';
import { useTranslation } from 'react-i18next';
import React from 'react';

let appVersion = '1.0.2'

export default function About() {
    const { t } = useTranslation();

    return (
        <div className="flex flex-col gap-3">
            <img
                src='appicon.png'
                className='mx-auto h-[100px] mb-[5px]'
                draggable={false}
            />
            <div className='content-center'>
                <h1 className='font-bold text-2xl text-center'>Handy-Translate</h1>
                <p className='text-center text-sm text-gray-500 mb-[5px]'>{appVersion}</p>
                <p className='text-center text-sm text-gray-500 mb-[5px]'>鼠标中键或者Ctrl+c+c唤醒应用</p>
                <p className='text-center text-sm text-gray-500 mb-[5px]'>Alt+Shift+Q OCR截图翻译</p>
                <Divider />
                <div className='flex justify-between'>
                    <Button
                        variant='light'
                        className='my-[5px]'
                        size='sm'
                        onPress={() => {
                            window.open('https://github.com/byzze/handy-translate');
                        }}
                    >
                        {t('config.about.github')}
                    </Button>
                    <Popover
                        placement='top'
                        offset={10}
                    >
                        <PopoverTrigger>
                            <Button
                                variant='light'
                                className='my-[5px]'
                                size='sm'
                            >
                                {t('config.about.feedback')}
                            </Button>
                        </PopoverTrigger>
                        <PopoverContent>
                            <div className='flex justify-between'>
                                <Button
                                    variant='light'
                                    className='my-[5px]'
                                    size='sm'
                                    onPress={() => {
                                        window.open('https://github.com/byzze/handy-translate/issues');
                                    }}
                                >
                                    {t('config.about.issue')}
                                </Button>

                            </div>
                        </PopoverContent>
                    </Popover>
                    <Button
                        variant='light'
                        className='my-[5px]'
                        size='sm'
                        onPress={() => {
                            window.open('mailto:luoyd@163.com');
                        }}
                    >
                        {t('config.about.email')} luoyd@163.com
                    </Button>
                </div>
                <Divider />
            </div>
        </div>
    );
}
