const { chromium } = await import(process.env.ARENALAB_PLAYWRIGHT_MODULE || 'playwright');
import { mkdir, writeFile } from 'node:fs/promises';
import assert from 'node:assert/strict';
const root=process.env.ARENALAB_BROWSER_URL||'http://localhost:8080';
const destination='artifacts/raw/browser';await mkdir(destination,{recursive:true});
const browser=await chromium.launch({headless:true,...(process.env.ARENALAB_CHROMIUM?{executablePath:process.env.ARENALAB_CHROMIUM}:{})});
const contexts=[];const pages=[];const errors=[];const report={started:new Date().toISOString(),baseURL:root,journeys:[]};
try{
 for(let i=0;i<2;i++){const context=await browser.newContext({viewport:{width:1280,height:1000},recordVideo:{dir:destination}});contexts.push(context);await context.tracing.start({screenshots:true,snapshots:true});const page=await context.newPage();pages.push(page);page.on('pageerror',e=>errors.push(String(e)));await page.goto(root);await page.selectOption('#player',String(i));await page.click('#login');await page.waitForFunction(()=>document.querySelector('#status').textContent.includes('已登录'));}
 for(const mode of ['turn','dedicated']){
  for(const page of pages)await page.selectOption('#mode',mode);
  await Promise.all(pages.map(page=>page.click('#match')));
  for(const page of pages)await page.waitForFunction(()=>document.querySelector('#status').textContent==='running',{},{timeout:10000});
  if(mode==='turn'){
   for(const page of pages)await page.check('#auto');
   await pages[0].waitForFunction(()=>Number(document.querySelector('#version').textContent)>=8);
   await pages[0].click('#reconnect');
  }else{
   await pages[0].keyboard.down('d');await pages[1].keyboard.down('a');
   await pages[0].waitForFunction(()=>Number(document.querySelector('#version').textContent)>=40);
   await pages[0].keyboard.up('d');await pages[1].keyboard.up('a');
   await pages[0].click('#reconnect');
  }
  for(const page of pages)await page.waitForFunction(()=>document.querySelector('#status').textContent==='finished',{},{timeout:40000});
  const snapshots=await Promise.all(pages.map(page=>page.evaluate(()=>({version:document.querySelector('#version').textContent,match:document.querySelector('#match-id').textContent,winner:document.querySelector('#turn').textContent,score:document.querySelector('#score').textContent}))));
  assert.equal(snapshots[0].match,snapshots[1].match);assert.equal(snapshots[0].version,mode==='turn'?'41':'600');assert.equal(snapshots[0].version,snapshots[1].version);assert.equal(snapshots[0].winner,snapshots[1].winner);
  for(let i=0;i<2;i++)await pages[i].screenshot({path:`${destination}/${mode}-p${i}.png`,fullPage:true});report.journeys.push({mode,passed:true,snapshots});console.log(`PASS browser ${mode}: two real contexts, reconnect, terminal agreement`);
 }
 assert.deepEqual(errors,[]);report.passed=true;
}catch(e){report.passed=false;report.error=String(e);for(let i=0;i<pages.length;i++)await pages[i].screenshot({path:`${destination}/failure-p${i}.png`,fullPage:true}).catch(()=>{});throw e;
}finally{
 report.finished=new Date().toISOString();report.pageErrors=errors;await writeFile(`${destination}/report.json`,JSON.stringify(report,null,2));for(let i=0;i<contexts.length;i++){await contexts[i].tracing.stop({path:`${destination}/p${i}-trace.zip`});await contexts[i].close();}await browser.close();
}
