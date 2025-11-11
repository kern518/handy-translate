import { DragDropContext, Draggable, Droppable } from 'react-beautiful-dnd';
import { Spacer, ButtonGroup, Button, Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, useDisclosure } from '@nextui-org/react';
import { AiFillCloseCircle, AiFillMinusCircle, AiFillSetting } from 'react-icons/ai';
import React, { useState, useEffect } from 'react';
import { BsPinFill, BsTranslate, BsInfoSquareFill } from 'react-icons/bs';
import LanguageArea from './components/LanguageArea';
import SourceArea from './components/SourceArea';
import TargetArea from './components/TargetArea';
import Way from './components/Way';
import About from './components/About';

import { useConfig, useSyncAtom, useVoice, useToastStyle } from '../../hooks';
import { translateServiceListAtom } from './components/Way';
import { atom, useAtom, useAtomValue } from 'jotai';
import { GetTranslateWay } from '../../../bindings/handy-translate/app';
import { Window } from "@wailsio/runtime";

let blurTimeout = null;
let resizeTimeout = null;
let moveTimeout = null;
let osType = "Windows_NT"

export default function Translate({ variable, onUpdateVariable }) {

    const { isOpen, onOpen, onOpenChange } = useDisclosure();
    const { isOpen: isSettingOpen, onOpen: onSettingOpen, onOpenChange: onSettingOpenChange } = useDisclosure();
    const { isOpen: isTranslateOpen, onOpen: onTranslateOpen, onOpenChange: onTranslateOpenChange } = useDisclosure();
    const { isOpen: isAboutOpen, onOpen: onAboutOpen, onOpenChange: onAboutOpenChange } = useDisclosure();

    const [modalPlacement, setModalPlacement] = React.useState("auto");
    const [alwaysOnTop] = useConfig('translate_always_on_top', false);
    const [hideSource] = useConfig('hide_source', false);
    const [hideLanguage] = useConfig('hide_language', false);
    const [pined, setPined] = useState(false);
    const [translate, setTranslate] = useState(false);
    const [setting, setSetting] = useState(false);
    const [about, setAbout] = useState(false);
    const [serviceConfig, setServiceConfig] = useState(null);

    const [translateServiceList, setTranslateServiceList] = useAtom(translateServiceListAtom);

    useEffect(() => {
        GetTranslateWay().then(result => {
            setTranslateServiceList([result]);
        });
    }, []);

    const getServiceConfig = async () => {
        let config = {};
        for (const service of translateServiceList) {
            config[service] = {}
        }
        setServiceConfig({ ...config });
    };
    const onDragEnd = async (result) => {

    };

    useEffect(() => {
        if (translateServiceList !== null) {
            getServiceConfig();
        }
    }, [translateServiceList]);

    return (
        (
            <div
                className={`bg-background h-screen w-screen ${osType === 'Linux' && 'rounded-[10px] border-1 border-default-100'
                    }`}
            >
                <div
                    className='fixed top-[5px] left-[5px] right-[5px] h-[30px]'
                    style={{ '--webkit-app-region': 'drag' }}
                />
                <div className={`h-[35px] w-full flex ${osType === 'Darwin' ? 'justify-end' : 'justify-between'}`}>
                    <ButtonGroup className='mr-[5px]'>
                        <Button
                            isIconOnly
                            size='sm'
                            variant='flat'
                            disableAnimation
                            className='my-auto bg-transparent'
                            onPress={() => {
                                onAboutOpen();
                                setAbout(about);
                            }}
                        >
                            <BsInfoSquareFill className={`text-[20px] ${isAboutOpen ? 'text-primary' : 'text-default-400'}`} />
                        </Button>
                        <Button
                            isIconOnly
                            size='sm'
                            variant='flat'
                            disableAnimation
                            className='my-auto bg-transparent'
                            onPress={() => {
                                onTranslateOpen();
                                setTranslate(!translate);
                            }}
                        >
                            <BsTranslate className={`text-[20px] ${translate ? 'text-primary' : 'text-default-400'}`} />
                        </Button>
                    </ButtonGroup>

                    <ButtonGroup className='mr-[5px]'>
                        <Button
                            isIconOnly
                            size='sm'
                            variant='flat'
                            disableAnimation
                            className='my-auto bg-transparent'
                            onPress={() => {
                                if (pined) {
                                    Window.SetAlwaysOnTop(false);
                                } else {
                                    Window.SetAlwaysOnTop(true);
                                }
                                setPined(!pined);
                            }}
                        >
                            <BsPinFill className={`text-[20px] ${pined ? 'text-primary' : 'text-default-400'}`} />
                        </Button>
                        <Button
                            isIconOnly
                            size='sm'
                            variant='flat'
                            disableAnimation
                            className={`my-auto ${osType === 'Darwin' && 'hidden'} bg-transparent`}
                            onPress={() => {
                                Window.Hide();
                            }}
                        >
                            <AiFillCloseCircle className='text-[20px] text-default-400' />
                        </Button>
                    </ButtonGroup>

                    <Modal
                        isOpen={isAboutOpen}
                        placement={modalPlacement}
                        onOpenChange={() => {
                            onAboutOpenChange()
                            setAbout(!about);
                        }}
                    >
                        <ModalContent>
                            {(onClose) => (
                                <>
                                    <ModalHeader className="flex flex-col gap-1">关于应用</ModalHeader>
                                    <ModalBody>
                                        <About />
                                    </ModalBody>
                                    <ModalFooter>
                                        <Button color="danger" variant="light" onPress={() => {
                                            onClose()
                                        }} >
                                            取消
                                        </Button>
                                    </ModalFooter>
                                </>
                            )}
                        </ModalContent>
                    </Modal>

                    <Modal
                        isOpen={isTranslateOpen}
                        placement={modalPlacement}
                        onOpenChange={() => {
                            onTranslateOpenChange()
                            setTranslate(!translate);
                        }}
                    >
                        <ModalContent>
                            {(onClose) => (
                                <>
                                    <ModalHeader className="flex flex-col gap-1">翻译服务</ModalHeader>
                                    <ModalBody>
                                        <Way />
                                    </ModalBody>
                                    <ModalFooter>
                                        <Button color="danger" variant="light" onPress={() => {
                                            onClose()
                                        }} >
                                            取消
                                        </Button>
                                    </ModalFooter>
                                </>
                            )}
                        </ModalContent>
                    </Modal>

                    <Modal
                        isOpen={isOpen}
                        placement={modalPlacement}
                        onOpenChange={onOpenChange}
                    >
                        <ModalContent>
                            {(onClose) => (
                                <>
                                    <ModalHeader className="flex flex-col gap-1">退出</ModalHeader>
                                    <ModalBody>
                                        <p>
                                            确认退出翻译工具吗？
                                        </p>
                                    </ModalBody>
                                    <ModalFooter>
                                        <Button color="danger" variant="light" onPress={() => {
                                            onClose()
                                        }} >
                                            取消
                                        </Button>
                                        <Button color="primary" onPress={() => {
                                            Window.Close();
                                        }}>
                                            确认
                                        </Button>
                                    </ModalFooter>
                                </>
                            )}
                        </ModalContent>
                    </Modal>
                </div>
                <div className={`${osType === 'Linux' ? 'h-[calc(100vh-37px)]' : 'h-[calc(100vh-35px)]'} px-[8px]`}>
                    <div className='h-full overflow-y-auto'>
                        <div className={`${hideSource && 'hidden'}`}>
                            <SourceArea />
                            <Spacer y={2} />
                        </div>

                        <div className={`${hideLanguage && 'hidden'}`}>
                            <LanguageArea />
                            <Spacer y={2} />
                        </div>

                        <DragDropContext onDragEnd={onDragEnd}>
                            <Droppable
                                droppableId='droppable'
                                direction='vertical'
                            >
                                {(provided) => (
                                    <div
                                        ref={provided.innerRef}
                                        {...provided.droppableProps}
                                    >
                                        {translateServiceList !== null &&
                                            serviceConfig !== null &&
                                            translateServiceList.map((service, index) => {
                                                const config = serviceConfig[service] ?? {};
                                                const enable = config['enable'] ?? true;
                                                return enable ? (
                                                    <Draggable
                                                        key={service}
                                                        draggableId={service}
                                                        index={index}
                                                    >
                                                        {(provided) => (
                                                            <div
                                                                ref={provided.innerRef}
                                                                {...provided.draggableProps}
                                                            >
                                                                <TargetArea
                                                                    {...provided.dragHandleProps}
                                                                    name={service}
                                                                    index={index}
                                                                    translateServiceList={translateServiceList}
                                                                />
                                                                <Spacer y={2} />
                                                            </div>
                                                        )}
                                                    </Draggable>
                                                ) : (
                                                    <></>
                                                );
                                            })}
                                    </div>
                                )}
                            </Droppable>
                        </DragDropContext>
                    </div>
                </div>
            </div>
        )
    );
}
