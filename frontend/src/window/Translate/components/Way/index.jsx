import React from "react";
import { RadioGroup, Radio } from "@nextui-org/react";
import { useEffect } from "react";
import toast, { Toaster } from 'react-hot-toast';
import { atom, useAtom, useAtomValue } from 'jotai';

import { useSyncAtom } from '../../../../hooks';
import { GetTranslateMap, SetTranslateWay, GetTranslateWay } from '../../../../../bindings/handy-translate/app';
export const translateServiceListAtom = atom([]);

let timer = null;

export default function Way() {
    const [translateMap, setTranslateMap] = React.useState({});

    const [translateServiceList, setTranslateServiceList, syncTranslateServiceList] = useSyncAtom(translateServiceListAtom)

    const [selected, setSelected] = React.useState("");

    useEffect(() => {
        GetTranslateMap().then(result => {
            result = JSON.parse(result)
            setTranslateMap(result)
        })

        GetTranslateWay().then(result => {
            setSelected(result)
        })
    }, [])


    return (
        <div className="flex flex-col gap-3">
            <Toaster />
            <RadioGroup
                label="选择你想要的翻译服务"
                value={selected}
                onValueChange={(value => {
                    console.log(value)
                    setSelected(value)
                    SetTranslateWay(value)
                    setTranslateServiceList([value])

                    if (timer) {
                        clearTimeout(timer);
                    }

                    timer = setTimeout(() => {
                        syncTranslateServiceList()
                    }, 100);

                })}
            >
                {translateMap &&
                    Object.entries(translateMap).map((key) => {
                        return (<Radio key={key[0]} value={key[0]}>{key[1].name}</Radio>)
                    })
                }
            </RadioGroup>
        </div>
    );
}
